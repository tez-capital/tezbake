package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tez-capital/tezbake/cli"
)

func TestInstanceList(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "tezbake-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Save old BBdir
	oldBBDir := cli.BBdir
	defer func() { cli.BBdir = oldBBDir }()
	cli.BBdir = tmpDir

	// Case 1: No instances, no default (empty dir)
	output, err := ExecuteTest(t, RootCmd, "instance", "list")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "" && strings.TrimSpace(output) != "[]" {
		t.Errorf("Expected empty output or '[]', got '%s'", output)
	}

	// Case 2: Only default instance (other dir exists)
	os.Mkdir(filepath.Join(tmpDir, "other_data"), 0755)
	output, err = ExecuteTest(t, RootCmd, "instance", "list")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "default") {
		t.Errorf("Expected 'default' in output, got '%s'", output)
	}

	// Case 3: Named instances
	instancesDir := filepath.Join(tmpDir, "instances")
	os.Mkdir(instancesDir, 0755)
	os.Mkdir(filepath.Join(instancesDir, "instance1"), 0755)
	os.Mkdir(filepath.Join(instancesDir, "instance2"), 0755)

	output, err = ExecuteTest(t, RootCmd, "instance", "list")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "default") {
		t.Errorf("Expected 'default' in output (since other_data exists), got '%s'", output)
	}
	if !strings.Contains(output, "instance1") {
		t.Errorf("Expected 'instance1' in output, got '%s'", output)
	}
	if !strings.Contains(output, "instance2") {
		t.Errorf("Expected 'instance2' in output, got '%s'", output)
	}
}
