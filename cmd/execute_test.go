package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kballard/go-shellquote"
)

func TestExecuteCommandExitStatus(t *testing.T) {
	const commandEnv = "TEZBAKE_TEST_EXECUTE_COMMAND"
	if command := os.Getenv(commandEnv); command != "" {
		if err := executeCmd.Flags().Set("command", command); err != nil {
			t.Fatal(err)
		}
		executeCmd.Run(executeCmd, nil)
		t.Fatal("execute returned without exiting")
	}
	for _, tc := range []struct {
		command string
		code    int
		message string
	}{
		{"sh -c 'exit 0'", 0, ""},
		{"sh -c 'exit 7'", 7, ""},
		{shellquote.Join(filepath.Join(t.TempDir(), "missing")), 1, "Failed to execute command:"},
	} {
		child := exec.Command(os.Args[0], "-test.run=^TestExecuteCommandExitStatus$")
		child.Env = append(os.Environ(), commandEnv+"="+tc.command)
		output, err := child.CombinedOutput()
		if child.ProcessState == nil {
			t.Fatal(err)
		}
		if code := child.ProcessState.ExitCode(); code != tc.code || !strings.Contains(string(output), tc.message) {
			t.Fatalf("execute %q: code %d, output %q; want code %d and %q", tc.command, code, output, tc.code, tc.message)
		}
	}
}
