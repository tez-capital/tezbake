package pay

import (
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Tezpay) Upgrade(ctx *base.UpgradeContext, args ...string) (int, error) {
	wasRunning, _ := app.IsAnyServiceStatus("running")
	if wasRunning {
		exitcode, err := app.Stop()
		if err != nil {
			return exitcode, err
		}
	}
	err := base.UpdateAppDefinitionBranch(app.GetPath(), ctx.Branch, ctx.Version)
	if err != nil {
		return -1, err
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
