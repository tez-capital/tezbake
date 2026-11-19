package base

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hjson/hjson-go/v4"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"
)

type IAmiBasedApp interface {
	GetPath() string
}

func LoadAppDefinition(app IAmiBasedApp) (map[string]any, string, error) {
	def, path, err := ami.FindAppDefinition(app.GetPath())
	if err != nil {
		return nil, "", fmt.Errorf("failed to load '%s' definition (%s)", app.GetPath(), err.Error())
	}
	return def, path, nil
}

func LoadAppConfiguration(app IAmiBasedApp) (map[string]any, error) {
	def, _, err := LoadAppDefinition(app)
	if err != nil {
		return nil, fmt.Errorf("failed to load '%s' configuration (%s)", app.GetPath(), err.Error())
	}
	if config, ok := def["configuration"].(map[string]any); ok {
		return config, nil
	}
	return nil, fmt.Errorf("failed to load '%s' configuration - unexpected format", app.GetPath())
}

func GetActiveModel(app IAmiBasedApp) (map[string]any, error) {
	return ami.GetAppActiveModel(app.GetPath())
}

// try to a dumb conversion
// https://github.com/tez-capital/xtz.configs/blob/main/seoulnet.json
// to
// https://raw.githubusercontent.com/tez-capital/xtz.configs/refs/heads/main/seoulnet.json
func tryConvertGitHubContentURL(url string) string {
	if !strings.HasPrefix(url, "https://github.com/") || !strings.Contains(url, "/blob/") {
		return url
	}

	result := strings.Replace(url, "https://github.com/", "https://raw.githubusercontent.com/", 1)
	result = strings.Replace(result, "/blob/", "/refs/heads/", 1)
	return result
}

func setTypeBranchAndVersion(appDef map[string]any, branch string, version string) (hasChanged bool) {
	if branch == "" && version == "" {
		return false
	}
	typeMap, ok := appDef["type"].(map[string]any)
	if !ok {
		typeMap = make(map[string]any)
		if id, ok := appDef["type"].(string); ok {
			typeMap["id"] = id
			typeMap["version"] = "latest"
		}
		appDef["type"] = typeMap
	}

	if version != "" {
		origVersion, _ := typeMap["version"].(string)
		if origVersion != version {
			typeMap["version"] = version
			hasChanged = true
		}
	}

	if branch != "" {
		origID, _ := typeMap["id"].(string)

		baseID := origID
		if parts := strings.Split(origID, "."); len(parts) > 2 {
			baseID = strings.Join(parts[:len(parts)-1], ".")
		}

		var newID string
		if branch == "main" {
			newID = baseID
		} else {
			newID = fmt.Sprintf("%s.%s", baseID, branch)
		}

		if origID != newID {
			typeMap["id"] = newID
			hasChanged = true
		}
	}

	return hasChanged
}

func UpdateAppDefinitionBranch(appDir string, branch string, version string) error {
	if branch == "" && version == "" {
		return nil
	}

	appDef, err := ami.ReadAppDefinition(appDir, constants.DefaultAppJsonName)
	if err != nil {
		return fmt.Errorf("failed to read app definition: %w", err)
	}

	if hasChanged := setTypeBranchAndVersion(appDef, branch, version); !hasChanged {
		return nil
	}

	err = ami.WriteAppDefinition(appDir, appDef, constants.DefaultAppJsonName)
	if err != nil {
		return fmt.Errorf("failed to write app definition: %w", err)
	}

	return nil
}

func GenerateConfiguration(template map[string]any, ctx *SetupContext) (map[string]any, error) {
	appDef := template

	appDef["id"] = fmt.Sprintf("%s-%s", cli.BBInstanceId, appDef["id"])
	appDef["user"] = ctx.User
	setTypeBranchAndVersion(appDef, ctx.Branch, ctx.Version)

	appConfiguration := appDef["configuration"].(map[string]any)
	appCtxConfiguration := make(map[string]any)

	switch {
	case ctx.Configuration == "":
		return appDef, nil
	case util.IsValidUrl(ctx.Configuration):
		temporary, err := os.MkdirTemp("", "bb-configuration-*")
		if err != nil {
			return appDef, err
		}
		defer os.RemoveAll(temporary)
		tmpConfigurationFile := filepath.Join(temporary, "config.json")

		err = util.DownloadFile(tryConvertGitHubContentURL(ctx.Configuration), tmpConfigurationFile, false)
		if err != nil {
			return appDef, fmt.Errorf("failed to download configuration file - %s", ctx.Configuration)
		}
		configurationFileJson, err := os.ReadFile(tmpConfigurationFile)
		if err != nil {
			return appDef, fmt.Errorf("invalid configuration - %s (%s)", ctx.Configuration, err.Error())
		}

		configurationFile := make(map[string]any)
		err = hjson.Unmarshal(configurationFileJson, &configurationFile)
		if err != nil {
			return appDef, fmt.Errorf("invalid configuration - %s (%s)", ctx.Configuration, err.Error())
		}
		for k, v := range configurationFile {
			appConfiguration[k] = v
		}

		return appDef, nil
	default:
		err := hjson.Unmarshal([]byte(ctx.Configuration), &appCtxConfiguration)
		if err != nil {
			return appDef, fmt.Errorf("invalid configuration - %s (%s)", ctx.Configuration, err.Error())
		}
		for k, v := range appCtxConfiguration {
			appConfiguration[k] = v
		}
		return appDef, nil
	}
}
