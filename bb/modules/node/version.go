package bb_module_node

import (
	"encoding/json"

	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"
)

func (app *Node) GetVersions(options *ami.CollectVersionsOptions) (*ami.InstanceVersions, error) {
	var postprocess ami.RemoteVersionPostprocessFn = func(output string) (*ami.InstanceVersions, error) {
		bbCliVersions := &bb_module.BBInstanceVersions{}
		err := json.Unmarshal([]byte(output), bbCliVersions)
		if err != nil {
			return nil, err
		}
		result := &ami.InstanceVersions{
			Cli:      bbCliVersions.Cli,
			Packages: bbCliVersions.Node.Packages,
			Binaries: bbCliVersions.Node.Binaries,
			IsRemote: true,
		}
		return result, nil
	}
	return ami.GetVersions(app.GetPath(), options, &postprocess)
}

func (app *Node) GetAppVersion() (string, error) {
	return ami.GetAppVersion(app.GetPath())
}
