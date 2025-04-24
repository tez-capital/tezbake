package node

import (
	"fmt"
	"os"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	"github.com/AlecAivazis/survey/v2"
	log "github.com/sirupsen/logrus"
)

func promptReuseElevateCredentials() bool {
	var response bool
	prompt := &survey.Confirm{
		Message: "Do you want to reuse existing elevate credentials?",
	}
	err := survey.AskOne(prompt, &response)
	if err != nil {
		return false
	}
	return response
}

func (app *Node) GetSetupKind() string {
	return base.MergingSetupKind
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
				remoteElevatePassword := ""
				prompt := &survey.Password{
					Message: "Enter password to use for elevation on node remote:",
				}
				err = survey.AskOne(prompt, &remoteElevatePassword)
				util.AssertE(err, "Remote elevate requires password!")
				ctx.RemoteElevatePassword = remoteElevatePassword

				credentials := ctx.ToRemoteElevateCredentials()
				config.ElevationCredentials = credentials
				ami.WriteRemoteElevationCredentials(app.GetPath(), config, credentials)
			}
		}

		ami.WriteRemoteLocator(app.GetPath(), config, ctx.RemoteReset)
		err = ami.PrepareRemote(app.GetPath(), config, ctx.RemoteAuth)
		if err != nil {
			return -1, fmt.Errorf("failed to create remote node locator - %s", err.Error())
		}

		// TODO: remove with RemoteUser references
		// if ctx.RemoteUser != "" {
		// 	ctx.User = ctx.RemoteUser
		// }

		if ctx.RemoteElevate != ami.REMOTE_ELEVATION_NONE && locator != nil {
			// override with username we use to connect to remote
			// se we do not have to prompt for elevation when collecting info and other common tasks
			ctx.User = locator.Username
		}
	case app.IsRemoteApp():
		log.Warn("Found remote app locator. Setup will run on remote.")
		ami.SetupRemoteTezbake(app.GetPath(), "latest")
	}

	appDef, err := base.GenerateConfiguration(app.GetAmiTemplate(ctx), ctx)
	if err != nil {
		log.Warn(err)
	}
	oldAppDef, err := ami.ReadAppDefinition(app.GetPath(), constants.DefaultAppJsonName)
	if oldAppDef != nil && err == nil {
		if oldConfiguration, ok := (*oldAppDef)["configuration"].(map[string]interface{}); ok {
			log.Info("Found old configuration. Merging...")
			appDef["configuration"] = util.MergeMaps(oldConfiguration, appDef["configuration"].(map[string]interface{}), true)
		}
	}

	err = ami.WriteAppDefinition(app.GetPath(), appDef, constants.DefaultAppJsonName)
	if err != nil {
		return -1, fmt.Errorf("failed to write app definition - %s", err.Error())
	}

	return ami.SetupApp(app.GetPath(), args...)
}
