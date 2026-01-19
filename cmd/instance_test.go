package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
)

func TestInstanceCommand(t *testing.T) {
	// Setup a dummy command to check the path
	checkPathCmd := &cobra.Command{
		Use: "test-check-path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), cli.BBdir)
		},
	}
	RootCmd.AddCommand(checkPathCmd)
	defer RootCmd.RemoveCommand(checkPathCmd)

	alias := "my-instance"
	expectedPath := filepath.Join(constants.DefaultBBDirectory, "instances", alias)

	// Since RootCmd.Execute() is recursive and uses global state, we need to be careful.
	// We use ExecuteTest which captures output.
	// args will be ["instance", "my-instance", "test-check-path"]

	output, err := ExecuteTest(t, RootCmd, "instance", alias, "test-check-path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The output should be the path
	if !strings.Contains(output, expectedPath) {
		t.Errorf("Expected output to contain path '%s', got '%s'", expectedPath, output)
	}
}

func TestInstanceCommandAlias(t *testing.T) {
	// Setup a dummy command to check the path
	checkPathCmd := &cobra.Command{
		Use: "test-check-path-alias",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), cli.BBdir)
		},
	}
	RootCmd.AddCommand(checkPathCmd)
	defer RootCmd.RemoveCommand(checkPathCmd)

	alias := "alias-instance"
	expectedPath := filepath.Join(constants.DefaultBBDirectory, "instances", alias)

	// Test with 'i' alias
	output, err := ExecuteTest(t, RootCmd, "i", alias, "test-check-path-alias")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, expectedPath) {
		t.Errorf("Expected output to contain path '%s', got '%s'", expectedPath, output)
	}
}
