package bb_module_node

import (
	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"
	"alis.is/bb-cli/system"
)

func (app *Node) UpgradeStorage() (int, error) {
	upgradeStorageArgs := make([]string, 0)
	upgradeStorageArgs = append(upgradeStorageArgs, "node", "upgrade", "storage")
	exitCode, err := ami.Execute(app.GetPath(), upgradeStorageArgs...)
	if err != nil {
		return exitCode, err
	}
	return 0, nil
}

func (app *Node) Upgrade(ctx *bb_module.UpgradeContext, args ...string) (int, error) {
	isRemote, locator := ami.IsRemoteApp(app.GetPath())
	if isRemote {
		ami.PrepareRemote(app.GetPath(), locator, system.SSH_MODE_KEY)
	}

	wasRunning, _ := app.IsServiceStatus("node", "running")
	if !isRemote && wasRunning {
		exitCode, err := app.Stop()
		if err != nil {
			return exitCode, err
		}
	}
	exitCode, err := ami.SetupApp(app.GetPath(), args...)
	if err != nil {
		return exitCode, err
	}
	if ctx.UpgradeStorage {
		exitCode, err = app.UpgradeStorage()
		if err != nil {
			return exitCode, err
		}
	}

	if !isRemote && wasRunning {
		exitCode, err := app.Start()
		if err != nil {
			return exitCode, err
		}
	}
	return exitCode, err
}
