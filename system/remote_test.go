package system

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/kballard/go-shellquote"
)

func TestBuildUpEnvPreservesCredentials(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "expanded-password")
	password := "p a's\"\n$HOME;$(touch " + shellquote.Join(marker) + ")`printf substituted`"
	env := map[string]string{"ELEVATION_KIND": "sudo", "ELEVATION_PASSWORD": password}
	command := buildUpEnv(&env) + `printf '%s' "$ELEVATION_PASSWORD"`
	output, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil || string(output) != password {
		t.Fatalf("credentials were interpreted by the shell: output=%q error=%v", output, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("shell executed part of the password: %v", err)
	}
}

func TestForwardOutputLinesPreservesLargeLinesAndErrors(t *testing.T) {
	large := strings.Repeat("x", 128<<10)
	output := make(chan string, 3)
	if err := ForwardOutputLines(strings.NewReader(large+"\r\n\nlast"), output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{large, "", "last"} {
		if got := <-output; got != want {
			t.Fatalf("output line changed: got %d bytes, want %d", len(got), len(want))
		}
	}
	readErr := errors.New("output read failed")
	if err := ForwardOutputLines(iotest.ErrReader(readErr), output); !errors.Is(err, readErr) {
		t.Fatalf("output read error was lost: %v", err)
	}
}
