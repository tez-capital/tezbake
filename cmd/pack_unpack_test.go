package cmd

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePackMode(t *testing.T) {
	for _, mode := range []string{"full", "light"} {
		if _, err := validatePackMode(mode); err != nil {
			t.Fatalf("expected mode %q to be valid, got error: %v", mode, err)
		}
	}

	if _, err := validatePackMode("invalid"); err == nil {
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

	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "metadata.zip")

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create temp archive: %v", err)
	}

	writer := zip.NewWriter(file)
	raw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("failed to serialize metadata: %v", err)
	}
	if err := addBytesToArchive(writer, tezbakePackMetadataFile, raw, 0644); err != nil {
		t.Fatalf("failed to write metadata archive: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close metadata archive: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("failed to close temp archive file: %v", err)
	}

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("failed to open metadata archive: %v", err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
	})

	return &reader.Reader
}
