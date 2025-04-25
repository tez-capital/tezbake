package ami

import (
	"encoding/json"
	"fmt"
	"strings"
)

type AppDependencyVersions struct {
	Id      string `json:"id"`
	Version string `json:"version"`
}

type AppVersions struct {
	Binaries     map[string]string       `json:"binaries"`
	Dependencies []AppDependencyVersions `json:"dependencies"`
	Id           string                  `json:"id"`
	Version      string                  `json:"version"`
}
type InstanceVersions struct {
	Cli      string
	Binaries map[string]string
	Packages map[string]string
	IsRemote bool
}

type CollectVersionsOptions struct {
	Packages bool
	Binaries bool
}

type PackageVersionInfo struct {
	Dependencies []PackageVersionInfo
	Version      string
	Id           string
}

type RemoteVersionPostprocessFn func(string) (*InstanceVersions, error)

func GetVersions(workingDir string, options *CollectVersionsOptions, remotePostProcess *RemoteVersionPostprocessFn) (*InstanceVersions, error) {
	appVersionsRaw, exitCode, err := ExecuteGetOutput(workingDir, "--output-format=json", "version", "--all")
	if err != nil {
		return nil, fmt.Errorf("failed to get app versions (%s)", err.Error())
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("failed to get app versions - exit code %d", exitCode)
	}
	var appVersions AppVersions
	err = json.Unmarshal([]byte(appVersionsRaw), &appVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse app versions (%s)", err.Error())
	}

	isRemote, _ := IsRemoteApp(workingDir)
	if isRemote {
		//TODO: tezbake version
	}

	packages := make(map[string]string)
	if appVersions.Dependencies != nil {
		for _, v := range appVersions.Dependencies {
			if v.Id != "" {
				packages[v.Id] = v.Version
			}
		}
	}
	packages[appVersions.Id] = appVersions.Version

	binaries := appVersions.Binaries

	return &InstanceVersions{
		Binaries: binaries,
		Packages: packages,

		IsRemote: isRemote,
	}, nil

}

func GetAppVersion(workingDir string) (string, error) {
	appVersion, exitCode, err := ExecuteGetOutput(workingDir, "version")
	if err != nil {
		return "", fmt.Errorf("failed to get app version (%s)", err.Error())
	}
	if exitCode != 0 {
		return "", fmt.Errorf("failed to get app version - exit code %d", exitCode)
	}
	if appVersion == "" {
		return "", fmt.Errorf("failed to get app version - empty output")
	}

	return strings.Trim(appVersion, "\n"), nil
}
