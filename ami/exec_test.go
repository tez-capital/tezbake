package ami

import (
	"os/exec"
	"testing"
)

func TestApplyExistingWorkingDir(t *testing.T) {
	cmd := exec.Command("true")
	dir := t.TempDir()

	applyExistingWorkingDir(cmd, dir)

	if cmd.Dir != dir {
		t.Fatalf("expected working dir %q, got %q", dir, cmd.Dir)
	}
}

func TestApplyExistingWorkingDirSkipsMissingDir(t *testing.T) {
	cmd := exec.Command("true")

	applyExistingWorkingDir(cmd, "/definitely/missing/dir")

	if cmd.Dir != "" {
		t.Fatalf("expected empty working dir for missing path, got %q", cmd.Dir)
	}
}
