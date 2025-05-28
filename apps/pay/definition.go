package pay

import (
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Tezpay) LoadAppDefinition() (map[string]any, string, error) {
	return base.LoadAppDefinition(app)
}

func (app *Tezpay) LoadAppConfiguration() (map[string]any, error) {
	return base.LoadAppConfiguration(app)
}

func (app *Tezpay) GetActiveModel() (map[string]any, error) {
	return base.GetActiveModel(app)
}
