package ami

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
)

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

func collectPackageVersions(packageVersionInfo PackageVersionInfo, collected map[string]string) map[string]string {
	log.Trace("Collecting package version...")
	collected[packageVersionInfo.Id] = packageVersionInfo.Version
	collected = collectDependenciesVersions(packageVersionInfo.Dependencies, collected)
	return collected
}

func collectDependenciesVersions(packageVersionInfos []PackageVersionInfo, collected map[string]string) map[string]string {
	log.Trace("Collecting package dependencies...")
	for _, v := range packageVersionInfos {
		collected = collectPackageVersions(PackageVersionInfo(v), collected)
	}
	return collected
}

func getAllPackageVersions(workingDir string) map[string]string {
	versionTreeJson, err := os.ReadFile(path.Join(workingDir, ".version-tree.json"))
	if err != nil {
		return map[string]string{path.Base(workingDir): "missing version info"}
	}
	result := make(map[string]string)

	versionTree := PackageVersionInfo{}
	err = json.Unmarshal(versionTreeJson, &versionTree)
	if err != nil {
		log.Debug(err.Error())
		return result
	}
	result = collectPackageVersions(versionTree, result)
	return result
}

func getBinaryVersion(path string, homeDir string, version chan string) {
	binProc := exec.Command(path, "--version")
	log.Trace("Getting version of ", path)
	binProc.Env = append(binProc.Env, "HOME="+homeDir)
	if output, err := binProc.CombinedOutput(); err == nil {
		version <- strings.TrimSpace(string(output))
		return
	} else {
		log.Debug(err.Error())
	}
	version <- "unknown"
}

func getBinaryVersions(workingDir string) map[string]string {
	homeDir := path.Join(workingDir, "data")
	binDir := path.Join(workingDir, "bin")
	items, err := os.ReadDir(binDir)
	if err != nil {
		return map[string]string{}
	}

	resultChannels := make(map[string]chan string)
	for _, entry := range items {
		if !entry.IsDir() {
			resultChannels[entry.Name()] = make(chan string)
			go getBinaryVersion(path.Join(binDir, entry.Name()), homeDir, resultChannels[entry.Name()])
		}
	}

	result := make(map[string]string)
	for k, v := range resultChannels {
		result[k] = <-v
	}
	return result
}

type RemoteVersionPostprocessFn func(string) (*InstanceVersions, error)

func GetVersions(workingDir string, options *CollectVersionsOptions, remotePostProcess *RemoteVersionPostprocessFn) (*InstanceVersions, error) {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		var output string
		session, err := locator.OpenAppRemoteSessionS()
		if err == nil {
			output, _, err = session.ProxyToRemoteAppGetOutput()
		}

		var result *InstanceVersions
		result = nil
		if remotePostProcess != nil {
			remotePostProcessFn := *remotePostProcess
			result, err = remotePostProcessFn(output)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get remote app package versions (%s)", err.Error())
		}
		return result, nil
	}

	var packageVersions, binaryVersions map[string]string
	if options.Packages {
		packageVersions = getAllPackageVersions(workingDir)
	}
	if options.Binaries {
		binaryVersions = getBinaryVersions(workingDir)
	}
	return &InstanceVersions{
		Binaries: binaryVersions,
		Packages: packageVersions,
	}, nil
}

func GetAppVersion(workingDir string) (string, error) {
	log.Trace("Collecting version from '" + workingDir + "'...")
	var versionTreeJson []byte
	var err error
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		var versionTreeFile *sftp.File
		session, err := locator.OpenAppRemoteSessionS()
		if err == nil {
			versionTreeFile, err = session.sftpSession.Open(path.Join(locator.InstancePath, locator.App, ".version-tree.json"))
		}
		if err == nil {
			defer versionTreeFile.Close()
			versionTreeJson, _ = io.ReadAll(versionTreeFile)
		}
	} else {
		versionTreeJson, err = os.ReadFile(path.Join(workingDir, ".version-tree.json"))
	}
	versionTree := make(map[string]interface{})
	if err == nil {
		err = json.Unmarshal(versionTreeJson, &versionTree)
	}
	if err == nil {
		if version, ok := versionTree["version"].(string); ok {
			return version, err
		}
	}
	return "unknown", err
}
