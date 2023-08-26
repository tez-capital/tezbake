package bb_module_node

import (
	bb_module "alis.is/bb-cli/bb/modules"
)

func (app *Node) LoadAppDefinition() (map[string]interface{}, string, error) {
	return bb_module.LoadAppDefinition(app)
}
func (app *Node) LoadAppConfiguration() (map[string]interface{}, error) {
	return bb_module.LoadAppConfiguration(app)
}
