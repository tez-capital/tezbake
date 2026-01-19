package cmd

import (
	"strings"
	"testing"
)

func TestInstanceHelpContainsList(t *testing.T) {
	output, err := ExecuteTest(t, RootCmd, "instance", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "list") {
		t.Errorf("Expected help output to contain 'list', got:\n%s", output)
	}
	if !strings.Contains(output, "List available instances") {
		t.Errorf("Expected help output to contain 'List available instances', got:\n%s", output)
	}
}
