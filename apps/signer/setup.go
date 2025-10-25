package signer

import (
	"fmt"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	log "github.com/sirupsen/logrus"
)

func (app *Signer) Setup(ctx *base.SetupContext, args ...string) (int, error) {
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
	return ami.SetupApp(app.GetPath(), args...)
}
