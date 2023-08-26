package bb_module_node

import (
	"fmt"
	"os"

	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
)

func ContainsNewConfiguration(ctx *bb_module.SetupContext) bool {
	return ctx.Configuration != ""
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
			if err == nil && !ctx.RemoteReset {
				log.Info("Old remove locator found. Merging...")
				config.PopulateWith(locator)
			}
			err = os.MkdirAll(app.GetPath(), os.ModePerm)
			if err != nil {
				return -1, fmt.Errorf("failed to create directory structure for remote node locator - %s", err.Error())
			}
			ami.WriteRemoteElevationCredentials(app.GetPath(), config, ctx.ToRemoteElevateCredentials())
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
