package node

import (
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
)

func (app *Node) LoadAppDefinition() (map[string]interface{}, string, error) {
	return base.LoadAppDefinition(app)
}

func (app *Node) LoadAppConfiguration() (map[string]interface{}, error) {
	return base.LoadAppConfiguration(app)
}

func (app *Node) GetActiveModel() (map[string]interface{}, error) {
	return base.GetActiveModel(app)
}

func (app *Node) UpdateDalEndpoint(endpoint string) error {
	config, err := app.LoadAppConfiguration()
	if err != nil {
		return err
	}

	if endpoint != "" {
		config["DAL_NODE"] = endpoint
	} else {
		delete(config, "DAL_NODE")
	}

	return ami.UpdateAppConfiguration(app.GetPath(), config)
}
