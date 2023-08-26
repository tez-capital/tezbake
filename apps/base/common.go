package base

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/util"
)

type IAmiBasedApp interface {
	GetPath() string
}

func LoadAppDefinition(app IAmiBasedApp) (map[string]interface{}, string, error) {
	def, path, err := ami.FindAppDefinition(app.GetPath())
	if err != nil {
		return nil, "", fmt.Errorf("failed to load '%s' definition (%s)", app.GetPath(), err.Error())
	}
	return def, path, nil
}

func LoadAppConfiguration(app IAmiBasedApp) (map[string]interface{}, error) {
	def, _, err := LoadAppDefinition(app)
	if err != nil {
		return nil, fmt.Errorf("failed to load '%s' configuration (%s)", app.GetPath(), err.Error())
	}
	return def, nil
}

type BakeBuddyAppDefinition struct {
	Id      string
	Control BakeBuddyApp
}

func GenerateConfiguration(template map[string]interface{}, ctx *SetupContext) (map[string]interface{}, error) {
	appDef := template

	appDef["id"] = fmt.Sprintf("%s-%s", cli.BBInstanceId, appDef["id"])
	appDef["user"] = ctx.User
	appDef["type"].(map[string]interface{})["version"] = ctx.Version

	if ctx.Branch != "main" && ctx.Branch != "" {
		appDef["type"].(map[string]interface{})["id"] = fmt.Sprintf("%s.%s", appDef["type"].(map[string]interface{})["id"], ctx.Branch)
	}

	appConfiguration := appDef["configuration"].(map[string]interface{})
	appCtxConfiguration := make(map[string]interface{})
	if err := json.Unmarshal([]byte(ctx.Configuration), &appCtxConfiguration); err == nil || ctx.Configuration == "" {
		for k, v := range appCtxConfiguration {
			appConfiguration[k] = v
		}
		return appDef, nil
	}

	tmpConfigurationFile := path.Join(os.TempDir(), "bb-configuration")
	if util.IsValidUrl(ctx.Configuration) {
		err := util.DownloadFile(ctx.Configuration, tmpConfigurationFile, false)
		if err != nil {
			return appDef, fmt.Errorf("failed to download configuration file - %s", ctx.Configuration)
		}
		ctx.Configuration = tmpConfigurationFile
	}

	configurationFileJson, err := os.ReadFile(ctx.Configuration)
	if err != nil {
		return appDef, fmt.Errorf("invalid configuration - %s (%s)", ctx.Configuration, err.Error())
	}

	configurationFile := make(map[string]interface{})
	err = json.Unmarshal(configurationFileJson, &configurationFile)
	if err != nil {
		return appDef, fmt.Errorf("invalid configuration - %s (%s)", ctx.Configuration, err.Error())
	}
	for k, v := range configurationFile {
		appConfiguration[k] = v
	}

	return appDef, nil
}
