package system

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunElevatedProcessPreservesChildStatus(t *testing.T) {
	sudo := filepath.Join(t.TempDir(), "sudo")
	if err := os.WriteFile(sudo, []byte("#!/bin/sh\nexit \"$1\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, status := range []int{0, 1, 2, 22, 151} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			got, err := runElevatedProcess(exec.Command(sudo, strconv.Itoa(status)))
			if err != nil || got != status {
				t.Fatalf("sudo child status changed: got=%d want=%d error=%v", got, status, err)
			}
		})
	}
	status, err := runElevatedProcess(exec.Command(filepath.Join(t.TempDir(), "missing-sudo")))
	if err == nil || status != -1 {
		t.Fatalf("sudo launch failure was treated as a child exit: status=%d error=%v", status, err)
	}
}

func TestRequireElevatedUserSendsPasswordAndClosesInput(t *testing.T) {
	const childEnv = "TEZBAKE_TEST_ELEVATION_CHILD"
	if os.Getenv(childEnv) == "1" {
		RequireElevatedUser()
		t.Fatal("elevation returned without exiting")
	}
	if os.Geteuid() == 0 {
		t.Skip("already root; credential-based sudo is not used")
	}
	dir := t.TempDir()
	passwordFile := filepath.Join(dir, "password")
	// Reading to EOF exposes an unclosed pipe; a long password exposes writes before Start.
	if err := os.WriteFile(filepath.Join(dir, "sudo"), []byte("#!/bin/sh\ncat > \"$TEZBAKE_TEST_PASSWORD_FILE\"\nexit 23\n"), 0700); err != nil {
		t.Fatal(err)
	}
	password := strings.Repeat("p", 96<<10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRequireElevatedUserSendsPasswordAndClosesInput$")
	child.Env = append(os.Environ(), childEnv+"=1", "PATH="+dir+":"+os.Getenv("PATH"), "ELEVATION_KIND=sudo", "ELEVATION_PASSWORD="+password, "TEZBAKE_TEST_PASSWORD_FILE="+passwordFile)
	output, err := child.CombinedOutput()
	if ctx.Err() != nil || child.ProcessState == nil || child.ProcessState.ExitCode() != 23 {
		t.Fatalf("sudo password input or child exit failed: error=%v output=%q", err, output)
	}
	data, err := os.ReadFile(passwordFile)
	if err != nil || string(data) != password+"\n" {
		t.Fatalf("sudo password changed: got %d bytes error=%v", len(data), err)
	}
}
