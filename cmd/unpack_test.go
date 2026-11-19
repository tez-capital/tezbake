package cmd

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
)

type preflightRestoreApp struct {
	base.BakeBuddyApp
	id    string
	dir   string
	calls *int
}

func (a preflightRestoreApp) GetId() string        { return a.id }
func (a preflightRestoreApp) GetLocalPath() string { return filepath.Join(a.dir, a.id) }
func (a preflightRestoreApp) Unpack(_ string) (int, error) {
	*a.calls++
	return 0, nil
}

// stubInstalled replaces the installation check for the test.
func stubInstalled(t *testing.T, installed bool, err error) {
	t.Helper()
	old := isAppInstalled
	isAppInstalled = func(base.BakeBuddyApp) (bool, error) { return installed, err }
	t.Cleanup(func() { isAppInstalled = old })
}

func TestUnpackRequiresInstalledApps(t *testing.T) {
	root := t.TempDir()
	innerPath := writeTestNodeAppArchive(t, root)
	outerPath := filepath.Join(root, "backup.zip")
	outer, err := os.Create(outerPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(outer)
	if err := addFileToArchive(writer, "apps/node.zip", innerPath); err != nil {
		t.Fatal(err)
	}
	if err := addBytesToArchive(writer, tezbakePackMetadataFile, []byte(`{"version":1,"mode":"full","apps":[{"id":"node"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(writer.Close(), outer.Close()); err != nil {
		t.Fatal(err)
	}
	checkErr := errors.New("check failed")
	for name, tc := range map[string]struct {
		installed bool
		err       error
		want      string
	}{
		"not installed": {false, nil, "tezbake setup"},
		"check error":   {false, checkErr, "check failed"},
	} {
		t.Run(name, func(t *testing.T) {
			stubInstalled(t, tc.installed, tc.err)
			calls := 0
			old := apps.All
			apps.All = []base.BakeBuddyApp{preflightRestoreApp{id: "node", dir: root, calls: &calls}}
			t.Cleanup(func() { apps.All = old })
			err := unpackInstalledApps(unpackCmd, outerPath)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
			if calls != 0 {
				t.Fatalf("unpack mutated %d apps before preflight failed", calls)
			}
		})
	}
}

func TestUnpackRequiresDoubleBakeConfirmation(t *testing.T) {
	root := t.TempDir()
	innerPath := writeTestNodeAppArchive(t, root)
	outerPath := filepath.Join(root, "backup.zip")
	outer, err := os.Create(outerPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(outer)
	if err := addFileToArchive(writer, "apps/node.zip", innerPath); err != nil {
		t.Fatal(err)
	}
	if err := addBytesToArchive(writer, tezbakePackMetadataFile, []byte(`{"version":1,"mode":"full","apps":[{"id":"node"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(writer.Close(), outer.Close()); err != nil {
		t.Fatal(err)
	}
	stubInstalled(t, true, nil)
	oldInteractive := isInteractive
	isInteractive = func() bool { return false }
	t.Cleanup(func() { isInteractive = oldInteractive })
	for name, confirmed := range map[string]bool{"without --confirm": false, "with --confirm": true} {
		t.Run(name, func(t *testing.T) {
			if confirmed {
				if err := unpackCmd.Flags().Set(UnpackConfirmFlag, "true"); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { unpackCmd.Flags().Set(UnpackConfirmFlag, "false") })
			}
			calls := 0
			old := apps.All
			apps.All = []base.BakeBuddyApp{preflightRestoreApp{id: "node", dir: root, calls: &calls}}
			t.Cleanup(func() { apps.All = old })
			err := unpackInstalledApps(unpackCmd, outerPath)
			if confirmed && (err != nil || calls != 1) {
				t.Fatalf("confirmed unpack: calls=%d err=%v", calls, err)
			}
			if !confirmed && (!errors.Is(err, errUnpackCanceled) || calls != 0) {
				t.Fatalf("unconfirmed unpack without a terminal must abort before changes: calls=%d err=%v", calls, err)
			}
		})
	}
}

func writeTestNodeAppArchive(t *testing.T, root string) string {
	t.Helper()
	innerPath := filepath.Join(root, "node.zip")
	if err := writeNodeAppArchive(innerPath); err != nil {
		t.Fatal(err)
	}
	return innerPath
}

// writeNodeAppArchive writes the smallest node app archive that ValidateAppArchive accepts.
func writeNodeAppArchive(path string) error {
	inner, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(inner)
	for name, data := range map[string]string{
		"app.json":                   `{"id":"node","type":"xtz.node","user":"root","configuration":{}}`,
		"__ami_packed_metadata.json": `{"VERSION":1,"options":{}}`,
		"__xtz/remove-all.lua":       `return true`,
	} {
		entry, err := zw.Create(name)
		if err != nil {
			return errors.Join(err, inner.Close())
		}
		if _, err = entry.Write([]byte(data)); err != nil {
			return errors.Join(err, inner.Close())
		}
	}
	return errors.Join(zw.Close(), inner.Close())
}

func TestUnpackPreflightsEveryAppBeforeMutation(t *testing.T) {
	root := t.TempDir()
	innerPath := writeTestNodeAppArchive(t, root)
	for _, invalid := range []string{"missing signer", "node payload labeled as signer"} {
		t.Run(invalid, func(t *testing.T) {
			outerPath := filepath.Join(t.TempDir(), "backup.zip")
			outer, err := os.Create(outerPath)
			if err != nil {
				t.Fatal(err)
			}
			writer := zip.NewWriter(outer)
			if err := addFileToArchive(writer, "apps/node.zip", innerPath); err != nil {
				t.Fatal(err)
			}
			if invalid != "missing signer" {
				if err := addFileToArchive(writer, "apps/signer.zip", innerPath); err != nil {
					t.Fatal(err)
				}
			}
			metadata := []byte(`{"version":1,"mode":"full","apps":[{"id":"node"},{"id":"signer"}]}`)
			if err := addBytesToArchive(writer, tezbakePackMetadataFile, metadata, 0600); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := outer.Close(); err != nil {
				t.Fatal(err)
			}
			stubInstalled(t, true, nil)
			calls := 0
			old := apps.All
			apps.All = []base.BakeBuddyApp{preflightRestoreApp{id: "node", dir: root, calls: &calls}, preflightRestoreApp{id: "signer", dir: root, calls: &calls}}
			t.Cleanup(func() { apps.All = old })
			if err := unpackInstalledApps(unpackCmd, outerPath); err == nil {
				t.Fatal("accepted invalid signer archive")
			}
			if calls != 0 {
				t.Fatalf("unpack mutated %d apps before preflight failed", calls)
			}
		})
	}
}
