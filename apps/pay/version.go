package pay

import (
	"github.com/tez-capital/tezbake/ami"
)

func (app *Tezpay) GetVersions(options *ami.CollectVersionsOptions) (*ami.InstanceVersions, error) {
	return ami.GetVersions(app.GetPath(), options, nil)
}

func (app *Tezpay) GetAppVersion() (string, error) {
	return ami.GetAppVersion(app.GetPath())
}
