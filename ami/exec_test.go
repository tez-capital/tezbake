package ami

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func requireAmi(t *testing.T) {
	t.Helper()
	if _, _, err := GetEliAndAmiPath(); err != nil {
		t.Skip("eli and ami are required for this integration check")
	}
	t.Setenv("AMI_CACHE", t.TempDir())
	// AMI probes its repository at startup; keep this check offline.
	t.Setenv("https_proxy", "http://127.0.0.1:1")
}

func TestCreateAmiCmdLeavesPathResolutionToAmi(t *testing.T) {
	requireAmi(t)
	t.Chdir(t.TempDir())
	if err := os.Mkdir("node", 0755); err != nil {
		t.Fatal(err)
	}
	cmd, err := createAmiCmd("node", "--is-app-installed")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Dir != "" || !slices.Contains(cmd.Args, "--path=node") {
		t.Fatalf("relative path would resolve twice: Dir=%q Args=%q", cmd.Dir, cmd.Args)
	}
}

func TestIsAppInstalledSLocalRelativePath(t *testing.T) {
	requireAmi(t)
	t.Chdir(t.TempDir())
	app := filepath.Join("instance", "node")
	if err := os.MkdirAll(app, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(app, "app.json"), []byte(`{"type":"test.app"}`), 0644)
	writeTestFile(t, filepath.Join(app, ".version-tree.json"), []byte(`{"id":"test.app"}`), 0644)
	installed, err := IsAppInstalledS(app)
	if err != nil || !installed {
		t.Fatalf("relative app installation check: installed=%v error=%v", installed, err)
	}
	if err := os.Remove(filepath.Join(app, ".version-tree.json")); err != nil {
		t.Fatal(err)
	}
	installed, err = IsAppInstalledS(app)
	if err != nil || installed {
		t.Fatalf("absent version tree: installed=%v error=%v", installed, err)
	}
	installed, err = IsAppInstalledS("missing")
	if err != nil || installed {
		t.Fatalf("missing app: installed=%v error=%v", installed, err)
	}
}

func TestIsAppInstalledSRejectsBrokenLocator(t *testing.T) {
	app := t.TempDir()
	writeTestFile(t, filepath.Join(app, LocatorFile), []byte(`{broken`), 0644)
	if installed, err := IsAppInstalledS(app); installed || err == nil {
		t.Fatalf("broken locator was treated as an absent app: installed=%v error=%v", installed, err)
	}
}

func TestMalformedLocatorDoesNotExecuteOrWriteLocally(t *testing.T) {
	for _, kind := range []string{"malformed", "dangling symlink"} {
		t.Run(kind, func(t *testing.T) {
			app := t.TempDir()
			locatorPath := filepath.Join(app, LocatorFile)
			if kind == "malformed" {
				writeTestFile(t, locatorPath, []byte(`{broken`), 0600)
			} else if err := os.Symlink(filepath.Join(app, "missing"), locatorPath); err != nil {
				t.Fatal(err)
			}
			definition := []byte(`{"user":"previous"}`)
			writeTestFile(t, filepath.Join(app, "app.json"), definition, 0600)
			checks := map[string]func() error{
				"installation check": func() error { _, err := IsAppInstalledS(app); return err },
				"execute":            func() error { _, err := Execute(app, "info"); return err },
				"execute output":     func() error { _, _, err := ExecuteGetOutput(app, "info"); return err },
				"execute channel":    func() error { _, err := ExecuteWithOutputChannel(app, make(chan string, 2), "info"); return err },
				"find definition":    func() error { _, _, err := FindAppDefinition(app); return err },
				"read definition":    func() error { _, err := ReadAppDefinition(app, "app.json"); return err },
				"write definition":   func() error { return WriteAppDefinition(app, map[string]any{"user": "changed"}, "app.json") },
				"write file":         func() error { return WriteFile(app, []byte("changed"), "app.json") },
			}
			for name, check := range checks {
				t.Run(name, func(t *testing.T) {
					if err := check(); err == nil {
						t.Fatal("broken locator was treated as a local app")
					}
				})
			}
			data, err := os.ReadFile(filepath.Join(app, "app.json"))
			if err != nil || string(data) != string(definition) {
				t.Fatalf("local definition changed: data=%q error=%v", data, err)
			}
		})
	}
	if locator, err := loadAppLocator(t.TempDir()); locator != nil || !errors.Is(err, ErrNotRemoteApp) {
		t.Fatalf("absent locator was not treated as local: locator=%v error=%v", locator, err)
	}
}
