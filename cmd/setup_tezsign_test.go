package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/signer"
	"github.com/tez-capital/tezbake/constants"
)

func TestGetSignerServiceID(t *testing.T) {
	previous := apps.Signer
	t.Cleanup(func() { apps.Signer = previous })
	dir := t.TempDir()
	apps.Signer = signer.FromPath(dir)
	for _, backend := range []string{"tezsign", "octez", ""} {
		t.Run(backend, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(dir, "app.json"), []byte(`{"configuration":{"BACKEND":"`+backend+`"}}`), 0600); err != nil {
				t.Fatal(err)
			}
			want := constants.SignerAppServiceId
			if backend == "tezsign" {
				want = constants.TezsignAppServiceId
			}
			if got := getSignerServiceID(); got != want {
				t.Fatalf("backend %q: got service %q, want %q", backend, got, want)
			}
		})
	}
}
