package bb_module_node

import (
	"fmt"
	"os"

	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"
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
	return bb_module.MergingSetupKind
}

func (app *Node) Setup(ctx *bb_module.SetupContext, args ...string) (int, error) {
	if isRemote, _ := ami.IsRemoteApp(app.GetPath()); isRemote {
		log.Warn("Found remote node locator. Setup will run on remote.")
	}
	if !cli.IsRemoteInstance {
		if ctx.Remote != "" {
			locator, err := ami.LoadRemoteLocator(app.GetPath())
			config := ctx.ToRemoteConfiguration(app)
			useExistingCredentials := false
			if err == nil && !ctx.RemoteReset {
				log.Info("Old remoTe locator found. Merging...")
				config.PopulateWith(locator)
				useExistingCredentials = promptReuseElevateCredentials()
			}
			err = os.MkdirAll(app.GetPath(), os.ModePerm)
			if err != nil {
				return -1, fmt.Errorf("failed to create directory structure for remote node locator - %s", err.Error())
			}
			if !useExistingCredentials {
				remoteElevatePassword := ""
				prompt := &survey.Password{
					Message: "Enter password to use for elevation on remote:",
				}
				err = survey.AskOne(prompt, &remoteElevatePassword)
				util.AssertE(err, "Remote elevate requires password!")
				ctx.RemoteElevatePassword = remoteElevatePassword

				credentials := ctx.ToRemoteElevateCredentials()
				config.ElevationCredentials = credentials
				ami.WriteRemoteElevationCredentials(app.GetPath(), config, credentials)
			}

			ami.WriteRemoteLocator(app.GetPath(), config, ctx.RemoteReset)
			err = ami.PrepareRemote(app.GetPath(), config, ctx.RemoteAuth)
			if err != nil {
				return -1, fmt.Errorf("failed to create remote node locator - %s", err.Error())
			}
		}
	} else {
		if ctx.RemoteUser != "" {
			ctx.User = ctx.RemoteUser // remote user has priority if remote instance
		}
	}
	appDef, err := bb_module.GenerateConfiguration(app.GetAmiTemplate(ctx), ctx)
	if err != nil {
		log.Warn(err)
	}
	oldAppDef, err := ami.ReadAppDefinition(app.GetPath(), cli.DefaultAppJsonName)
	if oldAppDef != nil && err == nil {
		if oldConfiguration, ok := (*oldAppDef)["configuration"].(map[string]interface{}); ok {
			log.Info("Found old configuration. Merging...")
			appDef["configuration"] = util.MergeMaps(oldConfiguration, appDef["configuration"].(map[string]interface{}), true)
		}
	}

	err = ami.WriteAppDefinition(app.GetPath(), appDef, cli.DefaultAppJsonName)
	if err != nil {
		return -1, fmt.Errorf("failed to write app definition - %s", err.Error())
	}
	return ami.SetupApp(app.GetPath(), args...)
}
