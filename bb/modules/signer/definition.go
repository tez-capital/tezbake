package bb_module_signer

import (
	bb_module "alis.is/bb-cli/bb/modules"
)

func (app *Signer) LoadAppDefinition() (map[string]interface{}, string, error) {
	return bb_module.LoadAppDefinition(app)
}
func (app *Signer) LoadAppConfiguration() (map[string]interface{}, error) {
	return bb_module.LoadAppConfiguration(app)
}
