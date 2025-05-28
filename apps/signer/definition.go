package signer

import (
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Signer) LoadAppDefinition() (map[string]any, string, error) {
	return base.LoadAppDefinition(app)
}

func (app *Signer) LoadAppConfiguration() (map[string]any, error) {
	return base.LoadAppConfiguration(app)
}

func (app *Signer) GetActiveModel() (map[string]any, error) {
	return base.GetActiveModel(app)
}
