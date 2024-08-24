package peak

import (
	"fmt"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	log "github.com/sirupsen/logrus"
)

func (app *Peak) GetSetupKind() string {
	return base.MergingSetupKind
}

func (app *Peak) Setup(ctx *base.SetupContext, args ...string) (int, error) {
	appDef, err := base.GenerateConfiguration(app.GetAmiTemplate(ctx), ctx)
	if err != nil {
		log.Warn(err)
	}

	_, _ = app.Execute("autodetect-configuration") // we ignore the error here, user can always run autodetect-configuration manually

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
