package bb_module_signer

import (
	"alis.is/bb-cli/ami"
)

func (app *Signer) GetVersions(options *ami.CollectVersionsOptions) (*ami.InstanceVersions, error) {
	return ami.GetVersions(app.GetPath(), options, nil)
}

func (app *Signer) GetAppVersion() (string, error) {
	return ami.GetAppVersion(app.GetPath())
}
