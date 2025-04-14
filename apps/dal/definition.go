package dal

import (
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *DalNode) LoadAppDefinition() (map[string]interface{}, string, error) {
	return base.LoadAppDefinition(app)
}
func (app *DalNode) LoadAppConfiguration() (map[string]interface{}, error) {
	return base.LoadAppConfiguration(app)
}
