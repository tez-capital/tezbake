package base

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tez-capital/tezbake/ami"
)

type archiveFakeApp struct {
	BakeBuddyApp
	status    string
	statusErr error
	installed bool
	localPath string
	calls     []string
}

func (a *archiveFakeApp) GetId() string        { return "node" }
func (a *archiveFakeApp) GetLocalPath() string { return a.localPath }
func (a *archiveFakeApp) IsInstalled() bool    { return a.installed }
func (a *archiveFakeApp) GetServiceInfo() (map[string]AmiServiceInfo, error) {
	if a.statusErr != nil {
		return nil, a.statusErr
	}
	return map[string]AmiServiceInfo{"node": {Status: a.status}}, nil
}
func (a *archiveFakeApp) Stop(...string) (int, error) {
	a.calls = append(a.calls, "stop")
	return 0, nil
}
func (a *archiveFakeApp) Start(...string) (int, error) {
	a.calls = append(a.calls, "start")
	return 0, nil
}

func requireAmi(t *testing.T) {
	t.Helper()
	if _, _, err := ami.GetEliAndAmiPath(); err != nil {
		t.Skip("AMI integration test: " + err.Error())
	}
	t.Setenv("AMI_CACHE", t.TempDir())
	t.Setenv("https_proxy", "http://127.0.0.1:1")
}

// writeTargetDefinition makes dir look like an installed app with its own identity.
func writeTargetDefinition(t *testing.T, dir string) {
	t.Helper()
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	raw := "id: target\ntype: xtz.node\nuser: " + current.Username + "\nconfiguration: {}\n"
	if err := os.WriteFile(filepath.Join(dir, "app.hjson"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
}

func writeFixtureApp(t *testing.T, dir string) {
	t.Helper()
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		// restore chowns to this user, which must be possible without root
		"app.hjson": "id: fixture\ntype: xtz.node\nuser: " + current.Username + "\nconfiguration: {}\n",
		"ami.lua": `return {base="app",commands={
			pack={action=function(options) am.app.pack{destination=options.output,mode=options.light and "light" or "full"} end},
			setup={action=function() fs.write_file("configured", "yes") end}}}`,
		"data/file": "payload",
	}
	for name, data := range files {
		filename := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLocalPackUnpackRoundTripRestartsRunningApp(t *testing.T) {
	requireAmi(t)
	root := t.TempDir()
	source := filepath.Join(root, "source")
	writeFixtureApp(t, source)
	output := filepath.Join(root, "backup.zip")

	app := &archiveFakeApp{status: "running", localPath: source}
	if code, err := DefaultPack(app, &PackContext{Mode: PackModeFull}, output); err != nil || code != 0 {
		t.Fatalf("pack: code=%d err=%v", code, err)
	}
	if len(app.calls) != 2 || app.calls[0] != "stop" || app.calls[1] != "start" {
		t.Fatalf("running app was not stopped and restarted: %v", app.calls)
	}
	if err := ValidateAppArchive(output, "node"); err != nil {
		t.Fatalf("packed archive rejected: %v", err)
	}

	target := filepath.Join(root, "target")
	if err := os.MkdirAll(filepath.Join(target, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "data", "kept"), []byte("kept"), 0600); err != nil {
		t.Fatal(err)
	}
	writeTargetDefinition(t, target)
	app = &archiveFakeApp{status: "running", localPath: target}
	if code, err := DefaultUnpack(app, output); err != nil || code != 0 {
		t.Fatalf("unpack: code=%d err=%v", code, err)
	}
	if len(app.calls) != 2 || app.calls[0] != "stop" || app.calls[1] != "start" {
		t.Fatalf("running app was not stopped and restarted: %v", app.calls)
	}
	for name, want := range map[string]string{"data/file": "payload", "data/kept": "kept", "configured": "yes"} {
		got, err := os.ReadFile(filepath.Join(target, name))
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, err=%v", name, got, err)
		}
	}
	if _, err := os.Stat(filepath.Join(target, AmiPackedArchiveName)); !os.IsNotExist(err) {
		t.Fatalf("transport archive remains: %v", err)
	}
	definition, _, err := ami.FindAppDefinition(target)
	if err != nil || definition["id"] != "target" {
		t.Fatalf("installed id was not kept: %v, err=%v", definition["id"], err)
	}
}

func TestStoppedAppIsNotStarted(t *testing.T) {
	requireAmi(t)
	root := t.TempDir()
	source := filepath.Join(root, "source")
	writeFixtureApp(t, source)
	output := filepath.Join(root, "backup.zip")

	app := &archiveFakeApp{status: "stopped", localPath: source}
	if code, err := DefaultPack(app, nil, output); err != nil || code != 0 {
		t.Fatalf("pack: code=%d err=%v", code, err)
	}
	app.localPath = filepath.Join(root, "target")
	writeTargetDefinition(t, app.localPath)
	if code, err := DefaultUnpack(app, output); err != nil || code != 0 {
		t.Fatalf("unpack: code=%d err=%v", code, err)
	}
	if len(app.calls) != 0 {
		t.Fatalf("stopped app was stopped or started: %v", app.calls)
	}
}

func TestUnknownStatusBlocksPackAndUnpack(t *testing.T) {
	statusErr := errors.New("ami info failed")
	app := &archiveFakeApp{statusErr: statusErr, installed: true, localPath: t.TempDir()}
	if _, err := DefaultPack(app, nil, filepath.Join(t.TempDir(), "x.zip")); !errors.Is(err, statusErr) {
		t.Fatalf("pack ignored status error: %v", err)
	}
	writeTargetDefinition(t, app.localPath)
	if _, err := DefaultUnpack(app, filepath.Join(t.TempDir(), "x.zip")); !errors.Is(err, statusErr) {
		t.Fatalf("unpack ignored status error: %v", err)
	}
	if len(app.calls) != 0 {
		t.Fatalf("unexpected calls: %v", app.calls)
	}
}

func TestUnpackRequiresInstalledDefinition(t *testing.T) {
	app := &archiveFakeApp{status: "stopped", localPath: t.TempDir()}
	if _, err := DefaultUnpack(app, filepath.Join(t.TempDir(), "x.zip")); err == nil {
		t.Fatal("unpack onto a directory without an app definition succeeded")
	}
}

func TestUnpackFailureKeepsAppStopped(t *testing.T) {
	app := &archiveFakeApp{status: "running", localPath: t.TempDir()}
	writeTargetDefinition(t, app.localPath)
	if _, err := DefaultUnpack(app, filepath.Join(t.TempDir(), "missing.zip")); err == nil {
		t.Fatal("unpack of a missing archive succeeded")
	}
	if len(app.calls) != 1 || app.calls[0] != "stop" {
		t.Fatalf("failed restore should stop the app and leave it stopped: %v", app.calls)
	}
}

func TestRestoreIdentity(t *testing.T) {
	keys := []string{"REMOTE_NODE", "REMOTE_SSH_PORT"}
	restored := map[string]any{"id": "archived", "user": "archived-user", "configuration": map[string]any{
		"REMOTE_NODE": "archived@host", "REMOTE_SSH_PORT": "22", "OTHER": "archived",
	}}
	installed := map[string]any{"id": "target", "user": "target-user", "configuration": map[string]any{
		"REMOTE_NODE": "target@host", "OTHER": "target",
	}}
	restoreIdentity(restored, installed, keys)
	configuration := restored["configuration"].(map[string]any)
	if restored["id"] != "target" || restored["user"] != "target-user" {
		t.Fatalf("identity not kept: %v", restored)
	}
	if configuration["REMOTE_NODE"] != "target@host" {
		t.Fatalf("wiring key not copied: %v", configuration)
	}
	if _, ok := configuration["REMOTE_SSH_PORT"]; ok {
		t.Fatalf("wiring key missing on target must be deleted: %v", configuration)
	}
	if configuration["OTHER"] != "archived" {
		t.Fatalf("other configuration must come from the archive: %v", configuration)
	}

	// no wiring keys: configuration is untouched
	restored = map[string]any{"id": "archived", "user": "archived-user", "configuration": map[string]any{"REMOTE_NODE": "archived@host"}}
	restoreIdentity(restored, map[string]any{"id": "target"}, nil)
	if restored["configuration"].(map[string]any)["REMOTE_NODE"] != "archived@host" {
		t.Fatalf("configuration changed without wiring keys: %v", restored)
	}
	if _, ok := restored["user"]; ok {
		t.Fatalf("user missing on target must be deleted: %v", restored)
	}
}

func TestPackFailureStillRestartsApp(t *testing.T) {
	requireAmi(t)
	app := &archiveFakeApp{status: "running", localPath: t.TempDir()} // no app definition, pack fails
	if code, err := DefaultPack(app, nil, filepath.Join(t.TempDir(), "x.zip")); err == nil && code == 0 {
		t.Fatal("pack of empty directory succeeded")
	}
	if len(app.calls) != 2 || app.calls[1] != "start" {
		t.Fatalf("app not restarted after failed pack: %v", app.calls)
	}
}

func TestCopyFileReplacesSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err := os.WriteFile(source, []byte("archive"), 0644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, AmiPackedArchiveName)
	if err := os.Symlink(outside, destination); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(source, destination, 0600); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(outside); string(got) != "keep" {
		t.Fatalf("wrote through symlink: %q", got)
	}
	info, err := os.Lstat(destination)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		t.Fatalf("destination %v, %v", info, err)
	}
}

type appArchiveFixtureFile struct {
	name string
	data string
	mode os.FileMode
}

func TestValidateAppArchive(t *testing.T) {
	metadata := appArchiveFixtureFile{amiPackedMetadataFile, `{"VERSION":1,"options":[]}`, 0600}
	config := appArchiveFixtureFile{"app.json", `{"id":"fixture","type":"xtz.node","user":"root","configuration":{}}`, 0600}
	cleanup := appArchiveFixtureFile{"__xtz/remove-all.lua", "return", 0600}
	valid := []appArchiveFixtureFile{metadata, config, cleanup}
	tests := []struct {
		name  string
		files []appArchiveFixtureFile
		valid bool
	}{
		{"valid", valid, true},
		{"large data streams", append(valid, appArchiveFixtureFile{"data/chain", strings.Repeat("x", appArchiveConfigLimit+1), 0600}), true},
		{"object metadata options", []appArchiveFixtureFile{{amiPackedMetadataFile, `{"VERSION":1,"options":{}}`, 0600}, config, cleanup}, true},
		{"staged metadata source", []appArchiveFixtureFile{{amiPackedMetadataFile, `{"VERSION":1,"options":{"source":"app.zip"}}`, 0600}, config, cleanup}, true},
		{"external metadata source", []appArchiveFixtureFile{{amiPackedMetadataFile, `{"VERSION":1,"options":{"source":"/tmp/other.zip"}}`, 0600}, config}, false},
		{"traversing metadata source", []appArchiveFixtureFile{{amiPackedMetadataFile, `{"VERSION":1,"options":{"source":"../other.zip"}}`, 0600}, config}, false},
		{"nonstring metadata source", []appArchiveFixtureFile{{amiPackedMetadataFile, `{"VERSION":1,"options":{"source":{"path":"app.zip"}}}`, 0600}, config}, false},
		{"hjson preferred", append(valid, appArchiveFixtureFile{"app.hjson", "id: fixture\nuser: root\ntype: xtz.node\n", 0600}), true},
		{"missing configuration", []appArchiveFixtureFile{metadata}, false},
		{"missing metadata", []appArchiveFixtureFile{config}, false},
		{"invalid configuration", []appArchiveFixtureFile{metadata, {"app.json", "{broken", 0600}}, false},
		{"null configuration", []appArchiveFixtureFile{metadata, {"app.json", "null", 0600}}, false},
		{"missing id", []appArchiveFixtureFile{metadata, {"app.json", `{"type":"xtz.node"}`, 0600}, cleanup}, false},
		{"nonstring id", []appArchiveFixtureFile{metadata, {"app.json", `{"id":{},"type":"xtz.node"}`, 0600}, cleanup}, false},
		{"unsupported type", []appArchiveFixtureFile{metadata, {"app.json", `{"id":"fixture","type":"other.app"}`, 0600}, cleanup}, false},
		{"versioned object type", []appArchiveFixtureFile{metadata, {"app.json", `{"id":"fixture","type":{"id":"xtz.node.standard@1"}}`, 0600}, cleanup}, true},
		{"invalid preferred hjson", append(valid, appArchiveFixtureFile{"app.hjson", "{broken", 0600}), false},
		{"invalid metadata", []appArchiveFixtureFile{{amiPackedMetadataFile, "{broken", 0600}, config}, false},
		{"wrong AMI version", []appArchiveFixtureFile{{amiPackedMetadataFile, `{"VERSION":2,"options":{}}`, 0600}, config}, false},
		{"missing AMI options", []appArchiveFixtureFile{{amiPackedMetadataFile, `{"VERSION":1}`, 0600}, config}, false},
		{"invalid AMI options", []appArchiveFixtureFile{{amiPackedMetadataFile, `{"VERSION":1,"options":true}`, 0600}, config}, false},
		{"oversized metadata", []appArchiveFixtureFile{{amiPackedMetadataFile, strings.Repeat(" ", appArchiveConfigLimit+1), 0600}, config}, false},
		{"oversized configuration", []appArchiveFixtureFile{metadata, {"app.json", strings.Repeat(" ", appArchiveConfigLimit+1), 0600}}, false},
		{"traversal", append(valid, appArchiveFixtureFile{"../outside", "x", 0600}), false},
		{"nested traversal", append(valid, appArchiveFixtureFile{"data/../../outside", "x", 0600}), false},
		{"absolute", append(valid, appArchiveFixtureFile{"/outside", "x", 0600}), false},
		{"backslash", append(valid, appArchiveFixtureFile{`data\outside`, "x", 0600}), false},
		{"drive path", append(valid, appArchiveFixtureFile{"C:/outside", "x", 0600}), false},
		{"nul path", append(valid, appArchiveFixtureFile{"data/\x00outside", "x", 0600}), false},
		{"dot prefix", append(valid, appArchiveFixtureFile{"./data", "x", 0600}), false},
		{"dot root file", append(valid, appArchiveFixtureFile{".", "x", 0600}), false},
		{"dot root directory", append(valid, appArchiveFixtureFile{"./", "", os.ModeDir | 0700}), false},
		{"empty path component", append(valid, appArchiveFixtureFile{"data//file", "x", 0600}), false},
		{"duplicate", append(valid, config), false},
		{"symlink", append(valid, appArchiveFixtureFile{"linked", "../outside", os.ModeSymlink | 0777}), false},
		{"symlink directory", append(valid, appArchiveFixtureFile{"linked/", "", os.ModeSymlink | 0777}), false},
		{"device", append(valid, appArchiveFixtureFile{"device", "", os.ModeDevice | 0600}), false},
		{"file as parent", append(valid, appArchiveFixtureFile{"data/file", "x", 0600}, appArchiveFixtureFile{"data", "x", 0600}), false},
		{"file and directory collision", append(valid, appArchiveFixtureFile{"data/", "", os.ModeDir | 0700}, appArchiveFixtureFile{"data", "x", 0600}), false},
		{"directory without slash", append(valid, appArchiveFixtureFile{"data", "", os.ModeDir | 0700}), false},
		{"reserved archive", append(valid, appArchiveFixtureFile{AmiPackedArchiveName, "x", 0600}), false},
		{"reserved archive case variant", append(valid, appArchiveFixtureFile{"App.ZIP", "x", 0600}), false},
		{"reserved archive directory", append(valid, appArchiveFixtureFile{AmiPackedArchiveName + "/child", "x", 0600}), false},
		{"management locator", append(valid, appArchiveFixtureFile{ami.LocatorFile, `{}`, 0600}), false},
		{"management locator case variant", append(valid, appArchiveFixtureFile{"LOCATOR.JSON", `{}`, 0600}), false},
		{"management locator directory", append(valid, appArchiveFixtureFile{ami.LocatorFile + "/child", "x", 0600}), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archive := writeAppArchiveFixture(t, test.files)
			err := ValidateAppArchive(archive, "node")
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, error=%v", test.valid, err)
			}
		})
	}
}

func TestValidateAppArchiveBindsAppType(t *testing.T) {
	for _, app := range []struct {
		id, appType, cleanup string
	}{
		{"node", "xtz.node", "__xtz/remove-all.lua"},
		{"dal", "xtz.dal.standard@1", "__xtz/remove-all.lua"},
		{"signer", "xtz.signer@1", "__xtz/remove-all.lua"},
		{"peak", "tzc.tezpeak.standard", "__tezpeak/remove-all.lua"},
		{"pay", "tzc.tezpay", "__tezpay/remove-all.lua"},
	} {
		t.Run(app.id, func(t *testing.T) {
			archive := writeAppArchiveFixture(t, []appArchiveFixtureFile{
				{amiPackedMetadataFile, `{"VERSION":1,"options":[]}`, 0600},
				{"app.json", fmt.Sprintf(`{"id":"previous-instance-%s","type":{"id":%q},"user":"root"}`, app.id, app.appType), 0600},
				{app.cleanup, "return", 0600},
			})
			if err := ValidateAppArchive(archive, app.id); err != nil {
				t.Fatalf("rejected instance-prefixed app id: %v", err)
			}
			for _, expected := range []string{"node", "dal", "signer", "peak", "pay", ""} {
				if expected != app.id {
					if err := ValidateAppArchive(archive, expected); err == nil || !strings.Contains(err.Error(), "expected") {
						t.Fatalf("accepted %s payload in %q slot: %v", app.id, expected, err)
					}
				}
			}
		})
	}
}

func TestFullPackUsesNodeBlacklist(t *testing.T) {
	if _, _, err := ami.GetEliAndAmiPath(); err != nil {
		t.Skip("eli and ami are required for this integration check")
	}
	t.Setenv("AMI_CACHE", t.TempDir())
	t.Setenv("https_proxy", "http://127.0.0.1:1")
	appPath := t.TempDir()
	files := map[string]string{
		"app.hjson": "id: fixture\ntype: xtz.node\nuser: root\nconfiguration: {}\n",
		"ami.lua": `return {base="app",commands={pack={action=function(options)
			am.app.pack{destination=options.output,mode=options.light and "light" or "full",
			paths={"data/.tezos-node/store/**"}}
		end}}}`,
		"data/.tezos-node/store/chain": "chain data",
		"data/.tezos-node/config.json": "{}",
		"__xtz/remove-all.lua":         "return",
	}
	for name, data := range files {
		filename := filepath.Join(appPath, name)
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	archivePath := filepath.Join(t.TempDir(), "full.zip")
	if exitCode, err := ami.Execute(appPath, amiPackArgs(archivePath, PackModeFull)...); err != nil || exitCode != 0 {
		t.Fatalf("full pack failed: exit=%d error=%v", exitCode, err)
	}
	if err := ValidateAppArchive(archivePath, "node"); err != nil {
		t.Fatalf("real AMI archive rejected: %v", err)
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	foundChain, foundConfig := false, false
	for _, file := range archive.File {
		switch file.Name {
		case "data/.tezos-node/store/chain":
			foundChain = true
		case "data/.tezos-node/config.json":
			foundConfig = true
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if foundChain {
		t.Fatal("full pack included chain data; the app-specific blacklist was bypassed")
	}
	if !foundConfig {
		t.Fatal("full pack omitted node data outside the chain blacklist")
	}
	lightArchivePath := filepath.Join(t.TempDir(), "light.zip")
	if exitCode, err := ami.Execute(appPath, amiPackArgs(lightArchivePath, PackModeLight)...); err != nil || exitCode != 0 {
		t.Fatalf("light pack failed: exit=%d error=%v", exitCode, err)
	}
	if err := ValidateAppArchive(lightArchivePath, "node"); err != nil {
		t.Fatalf("real AMI light archive rejected: %v", err)
	}
}

func writeAppArchiveFixture(t *testing.T, files []appArchiveFixtureFile) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "app.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range files {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		header.SetMode(entry.mode)
		dst, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dst.Write([]byte(entry.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}
