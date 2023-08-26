package peak

import (
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Peak) LoadAppDefinition() (map[string]interface{}, string, error) {
	return base.LoadAppDefinition(app)
}
func (app *Peak) LoadAppConfiguration() (map[string]interface{}, error) {
	return base.LoadAppConfiguration(app)
}
