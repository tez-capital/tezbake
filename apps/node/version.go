package node

import (
	"encoding/json"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Node) GetVersions(options *ami.CollectVersionsOptions) (*ami.InstanceVersions, error) {
	var postprocess ami.RemoteVersionPostprocessFn = func(output string) (*ami.InstanceVersions, error) {
		bbCliVersions := &base.BBInstanceVersions{}
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
