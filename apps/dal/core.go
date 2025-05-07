package dal

import (
	"fmt"
	"path"
	"strings"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"

	log "github.com/sirupsen/logrus"
)

var (
	Id           string                 = constants.DalAppId
	AMI_TEMPLATE map[string]interface{} = map[string]interface{}{
		"id":   constants.DalAppId,
		"type": map[string]interface{}{"id": "xtz.dal", "version": "latest"},
		"configuration": map[string]interface{}{
			"NODE_ENDPOINT": "http://127.0.0.1:8732",
		},
		"user": "",
	}
)

type DalNode struct {
	Path string
}

// FromPath creates a new Node instance with the specified path.
// The path parameter is the directory path to be associated with the Node.
// If the path is empty, the default path will be used.
// It returns a pointer to the newly created Node instance.
func FromPath(path string) *DalNode {
	return &DalNode{
		Path: path,
	}
}

func (app *DalNode) GetPath() string {
	if app.Path != "" {
		return app.Path
	}
	return path.Join(cli.BBdir, Id)
}

func (app *DalNode) GetId() string {
	return strings.ToLower(constants.DalAppId)
}

func (app *DalNode) GetUser() string {
	if app.IsRemoteApp() {
		locator, err := ami.LoadRemoteLocator(app.GetPath())
		if err != nil {
			log.Warnf("Failed to load %s locator (%s)!", app.GetId(), err.Error())
			return ""
		}
		return locator.LocalUsername
	}

	def, _, err := base.LoadAppDefinition(app)
	if err != nil {
		log.Warnf("Failed to load %s definition (%s)!", app.GetId(), err.Error())
		return ""
	}
	if user, ok := def["user"].(string); ok {
		return user
	}
	return ""
}

func (app *DalNode) GetLabel() string {
	if isRemote, locator := ami.IsRemoteApp(app.GetPath()); isRemote {
		return strings.ToUpper(fmt.Sprintf("%s (REMOTE - %s:%s)", app.GetId(), locator.Host, locator.Port))
	}
	return strings.ToUpper(app.GetId())
}

func (app *DalNode) GetAmiTemplate(ctx *base.SetupContext) map[string]interface{} {
	return AMI_TEMPLATE
}

func (app *DalNode) IsRemoteApp() bool {
	isRemote, _ := ami.IsRemoteApp(app.GetPath())
	return isRemote
}

func (app *DalNode) IsInstalled() bool {
	return ami.IsAppInstalled(app.GetPath())
}

func (app *DalNode) SupportsRemote() bool {
	return true
}
