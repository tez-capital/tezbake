package peak

import (
	"github.com/tez-capital/tezbake/ami"
)

func (app *Peak) GetVersions(options ami.CollectVersionsOptions) (*ami.InstanceVersions, error) {
	return ami.GetVersions(app.GetPath(), options)
}

func (app *Peak) GetVersion() (string, error) {
	return ami.GetAppVersion(app.GetPath())
}
