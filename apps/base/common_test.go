package base

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetTypeBranchAndVersion(t *testing.T) {
	tests := []struct {
		name     string
		appDef   map[string]any
		branch   string
		version  string
		expected map[string]any
		changed  bool
	}{
		{
			name:   "sets branch on base type id",
			branch: "seoulnet",
			appDef: map[string]any{
				"type": map[string]any{"id": "xtz.node", "version": "latest"},
			},
			expected: map[string]any{
				"type": map[string]any{"id": "xtz.node.seoulnet", "version": "latest"},
			},
			changed: true,
		},
		{
			name:   "strips existing branch and sets new one",
			branch: "newnet",
			appDef: map[string]any{
				"type": map[string]any{"id": "xtz.node.oldnet", "version": "v1.0"},
			},
			expected: map[string]any{
				"type": map[string]any{"id": "xtz.node.newnet", "version": "v1.0"},
			},
			changed: true,
		},
		{
			name:   "strips branch when set to main",
			branch: "main",
			appDef: map[string]any{
				"type": map[string]any{"id": "xtz.node.seoulnet", "version": "v1.0"},
			},
			expected: map[string]any{
				"type": map[string]any{"id": "xtz.node", "version": "v1.0"},
			},
			changed: true,
		},
		{
			name:   "no change when branch already matches",
			branch: "seoulnet",
			appDef: map[string]any{
				"type": map[string]any{"id": "xtz.node.seoulnet", "version": "v1.0"},
			},
			expected: map[string]any{
				"type": map[string]any{"id": "xtz.node.seoulnet", "version": "v1.0"},
			},
			changed: false,
		},
		{
			name:    "sets version",
			version: "v2.0",
			appDef: map[string]any{
				"type": map[string]any{"id": "xtz.node", "version": "v1.0"},
			},
			expected: map[string]any{
				"type": map[string]any{"id": "xtz.node", "version": "v2.0"},
			},
			changed: true,
		},
		{
			name:   "sets both branch and version",
			branch: "seoulnet",
			appDef: map[string]any{
				"type": map[string]any{"id": "xtz.signer", "version": "latest"},
			},
			version: "v3.0",
			expected: map[string]any{
				"type": map[string]any{"id": "xtz.signer.seoulnet", "version": "v3.0"},
			},
			changed: true,
		},
		{
			name:   "creates type map if missing",
			branch: "testnet",
			appDef: map[string]any{},
			expected: map[string]any{
				"type": map[string]any{"id": ".testnet"},
			},
			changed: true,
		},
		{
			name:    "no change when branch and version are empty",
			branch:  "",
			version: "",
			appDef: map[string]any{
				"type": map[string]any{"id": "xtz.node", "version": "v1.0"},
			},
			expected: map[string]any{
				"type": map[string]any{"id": "xtz.node", "version": "v1.0"},
			},
			changed: false,
		},
		{
			name:   "no change when type.id already matches main (no branch to strip)",
			branch: "main",
			appDef: map[string]any{
				"type": map[string]any{"id": "xtz.node", "version": "v1.0"},
			},
			expected: map[string]any{
				"type": map[string]any{"id": "xtz.node", "version": "v1.0"},
			},
			changed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := setTypeBranchAndVersion(tt.appDef, tt.branch, tt.version)
			if changed != tt.changed {
				t.Errorf("hasChanged = %v, want %v", changed, tt.changed)
			}
			typeMap := tt.appDef["type"].(map[string]any)
			expectedType := tt.expected["type"].(map[string]any)
			for k, v := range expectedType {
				if typeMap[k] != v {
					t.Errorf("type[%q] = %v, want %v", k, typeMap[k], v)
				}
			}
		})
	}
}

func TestUpdateAppDefinitionBranch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string) error
		branch   string
		version  string
		validate func(t *testing.T, dir string)
		err      bool
	}{
		{
			name:   "writes updated app definition",
			branch: "seoulnet",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "app.json"), []byte("{\n\t\"type\": {\n\t\t\"id\": \"xtz.node\",\n\t\t\"version\": \"latest\"\n\t}\n}"), 0644)
			},
			validate: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, "app.json"))
				if err != nil {
					t.Fatal(err)
				}
				expected := "{\n\t\"type\": {\n\t\t\"id\": \"xtz.node.seoulnet\",\n\t\t\"version\": \"latest\"\n\t}\n}"
				if string(data) != expected {
					t.Errorf("got %q, want %q", string(data), expected)
				}
			},
		},
		{
			name:   "skips write when nothing changed",
			branch: "main",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "app.json"), []byte("{\n\t\"type\": {\n\t\t\"id\": \"xtz.node\",\n\t\t\"version\": \"v1.0\"\n\t}\n}"), 0644)
			},
			validate: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, "app.json"))
				if err != nil {
					t.Fatal(err)
				}
				expected := "{\n\t\"type\": {\n\t\t\"id\": \"xtz.node\",\n\t\t\"version\": \"v1.0\"\n\t}\n}"
				if string(data) != expected {
					t.Errorf("got %q, want %q", string(data), expected)
				}
			},
		},
		{
			name: "no-ops when both branch and version are empty",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "app.json"), []byte("{\n\t\"type\": {\n\t\t\"id\": \"xtz.node\",\n\t\t\"version\": \"v1.0\"\n\t}\n}"), 0644)
			},
			validate: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, "app.json"))
				if err != nil {
					t.Fatal(err)
				}
				expected := "{\n\t\"type\": {\n\t\t\"id\": \"xtz.node\",\n\t\t\"version\": \"v1.0\"\n\t}\n}"
				if string(data) != expected {
					t.Errorf("got %q, want %q", string(data), expected)
				}
			},
		},
		{
			name:   "errors when app definition not found",
			branch: "seoulnet",
			err:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.setup != nil {
				if err := tt.setup(dir); err != nil {
					t.Fatal(err)
				}
			}
			err := UpdateAppDefinitionBranch(dir, tt.branch, tt.version)
			if tt.err {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.validate(t, dir)
		})
	}
}
