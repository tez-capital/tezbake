package signer

import (
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Signer) LoadAppDefinition() (map[string]interface{}, string, error) {
	return base.LoadAppDefinition(app)
}

func (app *Signer) LoadAppConfiguration() (map[string]interface{}, error) {
	return base.LoadAppConfiguration(app)
}

func (app *Signer) GetActiveModel() (map[string]interface{}, error) {
	return base.GetActiveModel(app)
}
