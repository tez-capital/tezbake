package peak

import (
	"github.com/tez-capital/tezbake/ami"

	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Peak) Upgrade(ctx *base.UpgradeContext, args ...string) (int, error) {
	wasRunning, _ := app.IsServiceStatus("tezpeak", "running")
	if wasRunning {
		exitcode, err := app.Stop()
		if err != nil {
			return exitcode, err
		}
	}
	exitCode, err := ami.SetupApp(app.GetPath(), args...)
	if wasRunning {
		exitcode, err := app.Start()
		if err != nil {
			return exitcode, err
		}
	}
	return exitCode, err
}
