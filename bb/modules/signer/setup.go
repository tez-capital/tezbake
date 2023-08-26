package bb_module_signer

import (
	"fmt"

	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
)

func (app *Signer) GetSetupKind() string {
	return bb_module.MergingSetupKind
}

func (app *Signer) Setup(ctx *bb_module.SetupContext, args ...string) (int, error) {
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
