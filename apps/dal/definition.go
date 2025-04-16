package dal

import (
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *DalNode) LoadAppDefinition() (map[string]interface{}, string, error) {
	return base.LoadAppDefinition(app)
}

func (app *DalNode) LoadAppConfiguration() (map[string]interface{}, error) {
	return base.LoadAppConfiguration(app)
}

func (app *DalNode) GetActiveModel() (map[string]interface{}, error) {
	return base.GetActiveModel(app)
}

func (app *DalNode) UpdateNodeEndpoint(endpoint string) error {
	config, err := app.LoadAppConfiguration()
	if err != nil {
		return err
	}

	config["NODE_ENDPOINT"] = endpoint
	return ami.UpdateAppConfiguration(app.GetPath(), config)
}
