package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
)

func TestValidatePackMode(t *testing.T) {
	for _, mode := range []string{"full", "light"} {
		if err := base.ValidatePackMode(mode); err != nil {
			t.Fatalf("expected mode %q to be valid, got error: %v", mode, err)
		}
	}

	if err := base.ValidatePackMode("invalid"); err == nil {
		t.Fatal("expected invalid mode to fail validation")
	}
}

func TestLoadTezbakePackMetadataRejectsUnknownApp(t *testing.T) {
	reader := createPackMetadataArchive(t, tezbakePackMetadata{
		Version: tezbakePackFormatVersion,
		Mode:    "full",
		Apps: []tezbakePackedApp{
			{Id: "unknown", Remote: false},
		},
	})

	if _, err := loadTezbakePackMetadata(reader); err == nil {
		t.Fatal("expected unknown app id to be rejected")
	}
}

func TestLoadTezbakePackMetadataRejectsDuplicateApps(t *testing.T) {
	reader := createPackMetadataArchive(t, tezbakePackMetadata{
		Version: tezbakePackFormatVersion,
		Mode:    "light",
		Apps: []tezbakePackedApp{
			{Id: "node", Remote: true},
			{Id: "node", Remote: false},
		},
	})

	if _, err := loadTezbakePackMetadata(reader); err == nil {
		t.Fatal("expected duplicate app ids to be rejected")
	}
}

func TestLoadTezbakePackMetadataRejectsEmptyArchive(t *testing.T) {
	reader := createPackMetadataArchive(t, tezbakePackMetadata{Version: tezbakePackFormatVersion, Mode: "full"})
	if _, err := loadTezbakePackMetadata(reader); err == nil {
		t.Fatal("expected an archive with no apps to be rejected")
	}
}

type packTestApp struct {
	base.BakeBuddyApp
	dir    string
	err    error
	remote bool
}

func (app packTestApp) GetId() string        { return "node" }
func (app packTestApp) GetLocalPath() string { return app.dir }
func (app packTestApp) IsRemoteApp() bool    { return app.remote }
func (app packTestApp) Pack(_ *base.PackContext, output string) (int, error) {
	if app.err != nil {
		return 1, app.err
	}
	return 0, os.WriteFile(output, []byte("app archive"), 0600)
}

func TestPackPreservesPreviousBackupOnFailure(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "backup.zip")
	previous := []byte("previous backup")
	if err := os.WriteFile(output, previous, 0644); err != nil {
		t.Fatal(err)
	}
	app := packTestApp{dir: t.TempDir(), err: errors.New("pack failed")}
	if err := packApps(output, base.PackModeFull, []base.BakeBuddyApp{app}); err == nil {
		t.Fatal("expected pack to fail")
	}
	got, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(got, previous) {
		t.Fatalf("previous backup changed: got %q, error %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary pack output was not removed: %v, error %v", entries, err)
	}
}

func TestPackDoesNotArchiveRemoteCredentials(t *testing.T) {
	output := filepath.Join(t.TempDir(), "backup.zip")
	app := packTestApp{dir: t.TempDir(), remote: true}
	if err := packApps(output, base.PackModeFull, []base.BakeBuddyApp{app}); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		if strings.HasPrefix(file.Name, "remote/") {
			t.Fatalf("remote credentials were archived: %s", file.Name)
		}
	}
	if len(archive.File) != 2 {
		t.Fatalf("expected app archive and metadata, got %d entries", len(archive.File))
	}
}

func TestPackRejectsOutputInsideApp(t *testing.T) {
	appDir := t.TempDir()
	configPath := filepath.Join(appDir, "app.json")
	previous := []byte("app configuration")
	if err := os.WriteFile(configPath, previous, 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "app")
	if err := os.Symlink(appDir, alias); err != nil {
		t.Fatal(err)
	}
	app := packTestApp{dir: appDir}
	for _, output := range []string{configPath, filepath.Join(alias, "app.json")} {
		if err := packApps(output, base.PackModeFull, []base.BakeBuddyApp{app}); err == nil {
			t.Fatalf("expected output inside app directory to be rejected: %s", output)
		}
	}
	got, err := os.ReadFile(configPath)
	if err != nil || !bytes.Equal(got, previous) {
		t.Fatalf("app configuration changed: got %q, error %v", got, err)
	}
}

func TestGetInstalledAppsForPackRejectsInvalidRemote(t *testing.T) {
	appDir := t.TempDir()
	locatorRaw, err := json.Marshal(ami.RemoteConfiguration{
		App: "node", Host: "example.invalid", InstancePath: "/bake-buddy",
		PublicKey: filepath.Join(appDir, "missing.pub"), PrivateKey: filepath.Join(appDir, "missing.key"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, ami.LocatorFile), locatorRaw, 0600); err != nil {
		t.Fatal(err)
	}
	previousApps := apps.All
	apps.All = []base.BakeBuddyApp{packTestApp{dir: appDir}}
	t.Cleanup(func() { apps.All = previousApps })
	if _, err := getInstalledAppsForPack(); err == nil {
		t.Fatal("expected the missing remote SSH key error to be returned")
	}
	apps.All = []base.BakeBuddyApp{packTestApp{dir: filepath.Join(t.TempDir(), "missing")}}
	installed, err := getInstalledAppsForPack()
	if err != nil || len(installed) != 0 {
		t.Fatalf("absent apps should be skipped: %v, error %v", installed, err)
	}
}

func TestReadArchiveFileRejectsInvalidEntries(t *testing.T) {
	for name, entries := range map[string][]archiveTestEntry{
		"missing":   nil,
		"duplicate": {{name: "entry", data: []byte("one"), mode: 0600}, {name: "entry", data: []byte("two"), mode: 0600}},
		"symlink":   {{name: "entry", data: []byte("target"), mode: os.ModeSymlink | 0600}},
		"oversized": {{name: "entry", data: make([]byte, archiveSmallFileLimit+1), mode: 0600}},
	} {
		t.Run(name, func(t *testing.T) {
			reader := createTestArchive(t, entries...)
			if _, _, err := readSmallArchiveFile(reader, "entry"); err == nil {
				t.Fatal("expected invalid archive entry to be rejected")
			}
		})
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestExtractArchiveFileStreamsLargePayload(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create("apps/node.zip")
	if err != nil {
		t.Fatal(err)
	}
	const size = 16 << 20
	if _, err := io.Copy(entry, io.LimitReader(zeroReader{}, size)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "node.zip")
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	if err := extractArchiveFile(reader, "apps/node.zip", destination, 0600); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 4<<20 {
		t.Fatalf("extracting %d bytes allocated %d bytes", size, allocated)
	}
	info, err := os.Stat(destination)
	if err != nil || info.Size() != size {
		t.Fatalf("unexpected extracted size: %v, error %v", info, err)
	}
}

func TestPackHelp(t *testing.T) {
	output, err := ExecuteTest(t, RootCmd, "pack", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Usage:\n  tezbake pack [flags]") {
		t.Fatalf("expected pack usage output, got:\n%s", output)
	}
	if !strings.Contains(output, "--output string") {
		t.Fatalf("expected pack help output, got:\n%s", output)
	}
	if strings.Contains(output, "--output-format string") {
		t.Fatalf("expected inherited output-format to be hidden from pack help, got:\n%s", output)
	}
}

func TestUnpackHelp(t *testing.T) {
	output, err := ExecuteTest(t, RootCmd, "unpack", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Usage:\n  tezbake unpack [flags]") {
		t.Fatalf("expected unpack usage output, got:\n%s", output)
	}
	if !strings.Contains(output, "--source string") {
		t.Fatalf("expected unpack help output, got:\n%s", output)
	}
	if strings.Contains(output, "--output-format string") {
		t.Fatalf("expected inherited output-format to be hidden from unpack help, got:\n%s", output)
	}
}

func createPackMetadataArchive(t *testing.T, metadata tezbakePackMetadata) *zip.Reader {
	t.Helper()
	raw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("failed to serialize metadata: %v", err)
	}
	return createTestArchive(t, archiveTestEntry{name: tezbakePackMetadataFile, data: raw, mode: 0644})
}

type archiveTestEntry struct {
	name string
	data []byte
	mode os.FileMode
}

func createTestArchive(t *testing.T, entries ...archiveTestEntry) *zip.Reader {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		if err := addBytesToArchive(writer, entry.name, entry.data, entry.mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return reader
}
