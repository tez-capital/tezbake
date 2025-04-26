package dal

import (
	"github.com/tez-capital/tezbake/ami"
)

func (app *DalNode) GetVersions(options ami.CollectVersionsOptions) (*ami.InstanceVersions, error) {
	return ami.GetVersions(app.GetPath(), options)
}

func (app *DalNode) GetVersion() (string, error) {
	return ami.GetAppVersion(app.GetPath())
}
