package signer

import (
	"github.com/tez-capital/tezbake/ami"
)

func (app *Signer) GetVersions(options ami.CollectVersionsOptions) (*ami.InstanceVersions, error) {
	return ami.GetVersions(app.GetPath(), options)
}

func (app *Signer) GetVersion() (string, error) {
	return ami.GetAppVersion(app.GetPath())
}
