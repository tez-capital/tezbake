package apps_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
)

func TestRemoteUpgradeStopsOnPreparationFailure(t *testing.T) {
	root := t.TempDir()
	// .invalid never resolves; REMOTE_PASS avoids the password prompt.
	t.Setenv("REMOTE_PASS", "unused")
	for _, app := range []base.BakeBuddyApp{apps.NodeFromPath(root), apps.DalNodeFromPath(root)} {
		t.Run(app.GetId(), func(t *testing.T) {
			dir := app.GetLocalPath()
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			locator, err := json.Marshal(ami.RemoteConfiguration{
				App: app.GetId(), Username: "test", Host: "example.invalid", Port: "22", InstancePath: "/bake-buddy",
				PrivateKey: filepath.Join(dir, "missing-key"), PublicKey: filepath.Join(dir, "missing-key.pub"),
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ami.LocatorFile), locator, 0600); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { ami.ForgetRemoteConfiguration(dir) })
			code, err := app.Upgrade(&base.UpgradeContext{})
			if code != -1 || err == nil || !strings.Contains(err.Error(), "failed to connect to remote") {
				t.Fatalf("upgrade continued after preparation failed: exit=%d error=%v", code, err)
			}
		})
	}
}
