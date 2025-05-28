package peak

import (
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Peak) LoadAppDefinition() (map[string]any, string, error) {
	return base.LoadAppDefinition(app)
}

func (app *Peak) LoadAppConfiguration() (map[string]any, error) {
	return base.LoadAppConfiguration(app)
}

func (app *Peak) GetActiveModel() (map[string]any, error) {
	return base.GetActiveModel(app)
}
