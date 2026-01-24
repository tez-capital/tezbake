package dal

import (
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"
)

func (app *DalNode) Upgrade(ctx *base.UpgradeContext, args ...string) (int, error) {
	isRemote, locator := ami.IsRemoteApp(app.GetPath())
	if isRemote {
		ami.PrepareRemote(app.GetPath(), locator, system.SSH_MODE_KEY)
	}

	wasRunning, _ := app.IsAnyServiceStatus("running")
	if wasRunning {
		exitCode, err := app.Stop()
		if err != nil {
			return exitCode, err
		}
	}
	exitCode, err := ami.SetupApp(app.GetPath(), args...)
	if err != nil {
		return exitCode, err
	}

	if isRemote {
		// we need to set permissions for remote apps
		// while apps set their permissions automatically during setup
		// remote apps need to set permissions manually as setup is run on remote
		user := app.GetUser()
		if user != "" {
			util.ChownR(user, app.GetPath())
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
