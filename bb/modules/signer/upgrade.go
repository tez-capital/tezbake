package bb_module_signer

import (
	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"
)

func (app *Signer) Upgrade(ctx *bb_module.UpgradeContext, args ...string) (int, error) {
	wasRunning, _ := app.IsServiceStatus("signer", "running")
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
