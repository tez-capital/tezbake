package pay

import (
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Tezpay) LoadAppDefinition() (map[string]interface{}, string, error) {
	return base.LoadAppDefinition(app)
}

func (app *Tezpay) LoadAppConfiguration() (map[string]interface{}, error) {
	return base.LoadAppConfiguration(app)
}

func (app *Tezpay) GetActiveModel() (map[string]interface{}, error) {
	return base.GetActiveModel(app)
}
