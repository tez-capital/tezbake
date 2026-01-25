package node

import (
	"fmt"
	"path"
	"strings"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/logging"
)

var (
	Id           string         = constants.NodeAppId
	AMI_TEMPLATE map[string]any = map[string]any{
		"id":   constants.NodeAppId,
		"type": map[string]any{"id": "xtz.node", "version": "latest"},
		"configuration": map[string]any{
			"NODE_TYPE": "baker",
		},
		"user": "",
	}
)

type Node struct {
	Path string
}

// FromPath creates a new Node instance with the specified path.
// The path parameter is the directory path to be associated with the Node.
// If the path is empty, the default path will be used.
// It returns a pointer to the newly created Node instance.
func FromPath(path string) *Node {
	return &Node{
		Path: path,
	}
}

func (app *Node) GetPath() string {
	appPath := path.Join(cli.BBdir, Id)
	if app.Path != "" {
		appPath = path.Join(app.Path, Id)
	}

	if isRemote, locator := ami.IsRemoteApp(appPath); isRemote {
		return path.Join(locator.InstancePath, locator.App)
	}

	return appPath
}

func (app *Node) GetId() string {
	return strings.ToLower(constants.NodeAppId)
}

func (app *Node) GetUser() string {
	if isRemote, locator := ami.IsRemoteApp(app.GetPath()); isRemote {
		return locator.LocalUsername
	}

	def, _, err := base.LoadAppDefinition(app)
	if err != nil {
		logging.Warnf("Failed to load %s definition (%s)!", app.GetId(), err.Error())
		return ""
	}
	if user, ok := def["user"].(string); ok {
		return user
	}
	return ""
}

func (app *Node) GetLabel() string {
	if isRemote, locator := ami.IsRemoteApp(app.GetPath()); isRemote {
		return strings.ToUpper(fmt.Sprintf("%s (REMOTE - %s:%s)", app.GetId(), locator.Host, locator.Port))
	}
	return strings.ToUpper(app.GetId())
}

func (app *Node) GetAmiTemplate(ctx *base.SetupContext) map[string]any {
	return AMI_TEMPLATE
}

func (app *Node) IsRemoteApp() bool {
	isRemote, _ := ami.IsRemoteApp(app.GetPath())
	return isRemote
}

func (app *Node) IsInstalled() bool {
	return ami.IsAppInstalled(app.GetPath())
}

func (app *Node) SupportsRemote() bool {
	return true
}
