package base

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tez-capital/tezbake/ami"
)

func TestConfigureRemoteLocatorElevationPrompt(t *testing.T) {
	for name, tc := range map[string]struct {
		elevate ami.RemoteElevationKind
		ask     bool
		prompts int
	}{
		"new sudo credentials": {ami.REMOTE_ELEVATION_SUDO, true, 1},
		"reused credentials":   {ami.REMOTE_ELEVATION_SUDO, false, 0},
		"no elevation":         {ami.REMOTE_ELEVATION_NONE, true, 0},
		"new su credentials":   {ami.REMOTE_ELEVATION_SU, true, 1},
	} {
		t.Run(name, func(t *testing.T) {
			appDir := t.TempDir()
			t.Cleanup(func() { ami.ForgetRemoteConfiguration(appDir) })
			prompts := 0
			old := promptElevation
			promptElevation = func(string) (string, string) { prompts++; return "sudo password", "" }
			t.Cleanup(func() { promptElevation = old })
			locator := &ami.RemoteConfiguration{App: "node", Host: "remote.test", Username: "baker", Port: "22",
				InstancePath: "/bake-buddy", Elevate: tc.elevate, ElevationCredentialsDirectory: appDir,
				PrivateKey: filepath.Join(appDir, "idkey"), PublicKey: filepath.Join(appDir, "idkey.pub")}

			// a missing auth key makes PrepareRemote fail before it connects
			_, err := ConfigureRemoteLocator(appDir, locator, true, tc.ask, "key:"+filepath.Join(appDir, "missing"))
			if err == nil {
				t.Fatal("expected remote preparation to fail without an auth key")
			}
			if prompts != tc.prompts {
				t.Fatalf("prompted %d times, want %d", prompts, tc.prompts)
			}
			_, statErr := os.Stat(filepath.Join(appDir, ami.ElevationCredentialsFile))
			if saved := statErr == nil; saved != (tc.prompts == 1) {
				t.Fatalf("credentials saved=%v, want %v", saved, tc.prompts == 1)
			}
			if tc.prompts == 1 && (locator.ElevationCredentials == nil || locator.ElevationCredentials.Password != "sudo password") {
				t.Fatalf("locator lacks the new credentials: %v", locator.ElevationCredentials)
			}
			// the locator and key are written before the remote is prepared
			for _, name := range []string{ami.LocatorFile, "idkey", "idkey.pub"} {
				if _, err := os.Stat(filepath.Join(appDir, name)); err != nil {
					t.Fatalf("missing %s: %v", name, err)
				}
			}
		})
	}
}
