package node

import (
	"fmt"
	"os"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/apps/signer"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	log "github.com/sirupsen/logrus"
)

func promptReuseElevateCredentials() bool {
	response, err := util.PromptConfirm("Do you want to reuse existing elevate credentials?", false)
	if err != nil {
		return false
	}
	return response
}

func (app *Node) Setup(ctx *base.SetupContext, args ...string) (int, error) {
	switch {
	case ctx.Remote != "":
		locator, err := ami.LoadRemoteLocator(app.GetPath())
		config := ctx.ToRemoteConfiguration(app)
		useExistingCredentials := false
		if err == nil && !ctx.RemoteReset {
			log.Info("Old node remote locator found. Merging...")
			config.PopulateWith(locator)
			useExistingCredentials = promptReuseElevateCredentials()
		}
		err = os.MkdirAll(app.GetPath(), os.ModePerm)
		if err != nil {
			return -1, fmt.Errorf("failed to create directory structure for remote node locator - %s", err.Error())
		}
		if !useExistingCredentials {
			switch ctx.RemoteElevate {
			case ami.REMOTE_ELEVATION_SU:
				fallthrough
			case ami.REMOTE_ELEVATION_SUDO:
				remoteElevatePassword, err := util.PromptPassword("Enter password to use for elevation on node remote:")
				util.AssertE(err, "Remote elevate requires password!")
				ctx.RemoteElevatePassword = remoteElevatePassword

				credentials := ctx.ToRemoteElevateCredentials()
				config.ElevationCredentials = credentials
				ami.WriteRemoteElevationCredentials(app.GetPath(), config, credentials)
			}
		}

		locator = ami.WriteRemoteLocator(app.GetPath(), config, ctx.RemoteReset)
		err = ami.PrepareRemote(app.GetPath(), config, ctx.RemoteAuth)
		if err != nil {
			return -1, fmt.Errorf("failed to create remote node locator - %s", err.Error())
		}

		// on remote we need to use locator username
		ctx.User = locator.Username
	case app.IsRemoteApp():
		log.Warn("Found remote app locator. Setup will run on remote.")
		ami.SetupRemoteTezbake(app.GetPath(), "latest")

		locator, err := ami.LoadRemoteLocator(app.GetPath())
		if err != nil {
			return -1, fmt.Errorf("failed to load remote locator - %s", err.Error())
		}
		// on remote we need to use locator username
		ctx.User = locator.Username

		// patch missing local_username
		// TODO: remove this after October 2025
		// everyone should use local_username at that point
		if locator.LocalUsername == "" {
			definition, _, err := signer.FromPath("").LoadAppDefinition()
			if err != nil {
				return -1, fmt.Errorf("failed to load signer definition - %s", err.Error())
			}
			if user, ok := definition["user"].(string); ok {
				locator.LocalUsername = user
			} else {
				return -1, fmt.Errorf("failed to load signer definition - unexpected format")
			}
			ami.WriteRemoteLocator(app.GetPath(), locator, false)
		}
	}

	appDef, err := base.GenerateConfiguration(app.GetAmiTemplate(ctx), ctx)
	if err != nil {
		return -1, fmt.Errorf("failed to generate configuration - %s", err.Error())
	}
	oldAppDef, err := ami.ReadAppDefinition(app.GetPath(), constants.DefaultAppJsonName)
	if oldAppDef != nil && err == nil {
		if oldConfiguration, ok := oldAppDef["configuration"].(map[string]any); ok {
			log.Info("Found old configuration. Merging...")
			appDef["configuration"] = util.MergeMapsDeep(oldConfiguration, appDef["configuration"].(map[string]any), true)
		}
	}

	err = ami.WriteAppDefinition(app.GetPath(), appDef, constants.DefaultAppJsonName)
	if err != nil {
		return -1, fmt.Errorf("failed to write app definition - %s", err.Error())
	}

	exitCode, err := ami.SetupApp(app.GetPath(), args...)
	if err != nil || exitCode != 0 {
		return exitCode, err
	}

	if app.IsRemoteApp() {
		fmt.Println("------------> wtf wtf wtf <------------")
		// we need to set permissions for remote apps
		// while apps set their permissions automatically during setup
		// remote apps need to set permissions manually as setup is run on remote
		user := app.GetUser()
		if user != "" {
			util.ChownR(user, app.GetPath())
		}
	}
	return 0, nil
}
