package signer

import (
	"fmt"

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

func (app *Signer) GetBackend() (string, error) {
	configuration, err := app.LoadAppConfiguration()
	if err != nil {
		return "", err
	}

	// Safely check if the key exists and is a string
	backend, ok := configuration["BACKEND"].(string)
	if !ok {
		return "", fmt.Errorf("backend configuration is missing or not a valid string")
	}

	return backend, nil
}
