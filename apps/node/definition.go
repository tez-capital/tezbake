package node

import (
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Node) LoadAppDefinition() (map[string]interface{}, string, error) {
	return base.LoadAppDefinition(app)
}
func (app *Node) LoadAppConfiguration() (map[string]interface{}, error) {
	return base.LoadAppConfiguration(app)
}
