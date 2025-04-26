package node

import (
	"github.com/tez-capital/tezbake/ami"
)

func (app *Node) GetVersions(options ami.CollectVersionsOptions) (*ami.InstanceVersions, error) {
	return ami.GetVersions(app.GetPath(), options)
}

func (app *Node) GetVersion() (string, error) {
	return ami.GetAppVersion(app.GetPath())
}
