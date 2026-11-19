package util

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGetAppCommandStringFlagS(t *testing.T) {
	tests := []struct {
		name       string
		appId      string
		suffix     string
		globalFlag string
		setupFlags func(*cobra.Command)
		expected   string
	}{
		{
			name:       "returns per-app flag when set",
			appId:      "node",
			suffix:     "branch",
			globalFlag: "branch",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().String("node-branch", "", "")
				cmd.Flags().String("branch", "main", "")
				cmd.Flags().Set("node-branch", "seoulnet")
			},
			expected: "seoulnet",
		},
		{
			name:       "falls back to global flag when per-app is empty",
			appId:      "node",
			suffix:     "branch",
			globalFlag: "branch",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().String("node-branch", "", "")
				cmd.Flags().String("branch", "main", "")
			},
			expected: "main",
		},
		{
			name:       "returns empty when per-app is empty and global flag is empty string",
			appId:      "node",
			suffix:     "branch",
			globalFlag: "",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().String("node-branch", "", "")
			},
			expected: "",
		},
		{
			name:       "returns per-app value even when global flag is set",
			appId:      "signer",
			suffix:     "version",
			globalFlag: "version",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().String("signer-version", "", "")
				cmd.Flags().String("version", "", "")
				cmd.Flags().Set("signer-version", "v2.0")
				cmd.Flags().Set("version", "v1.0")
			},
			expected: "v2.0",
		},
		{
			name:       "returns empty when neither flag exists",
			appId:      "node",
			suffix:     "branch",
			globalFlag: "branch",
			setupFlags: func(cmd *cobra.Command) {},
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			tt.setupFlags(cmd)
			result := GetAppCommandStringFlagS(cmd, tt.appId, tt.suffix, tt.globalFlag)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
