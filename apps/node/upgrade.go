package node

import (
	"path"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"
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

func (app *Node) Upgrade(ctx *base.UpgradeContext, args ...string) (int, error) {
	isRemote, locator := ami.IsRemoteApp(app.GetPath())
	if isRemote {
		ami.PrepareRemote(app.GetPath(), locator, system.SSH_MODE_KEY)
	}

	wasRunning, _ := app.IsServiceStatus(constants.NodeAppServiceId, "running")
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

	if isRemote {
		// we need to set permissions for remote apps
		// while apps set their permissions automatically during setup
		// remote apps need to set permissions manually as setup is run on remote
		user := app.GetUser()
		if user != "" {
			util.ChownR(user, path.Join(app.GetPath()))
		}
	}

	if wasRunning {
		exitCode, err := app.Start()
		if err != nil {
			return exitCode, err
		}
	}
	return exitCode, err
}
