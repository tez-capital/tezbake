package node

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
	Id           string                 = constants.NodeAppId
	AMI_TEMPLATE map[string]interface{} = map[string]interface{}{
		"id":   constants.NodeAppId,
		"type": map[string]interface{}{"id": "xtz.node", "version": "latest"},
		"configuration": map[string]interface{}{
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
	if app.Path != "" {
		return app.Path
	}
	return path.Join(cli.BBdir, Id)
}

func (app *Node) GetId() string {
	return strings.ToLower(constants.NodeAppId)
}

func (app *Node) GetLabel() string {
	if isRemote, locator := ami.IsRemoteApp(app.GetPath()); isRemote {
		return strings.ToUpper(fmt.Sprintf("%s (REMOTE - %s:%s)", app.GetId(), locator.Host, locator.Port))
	}
	return strings.ToUpper(app.GetId())
}

func (app *Node) GetAmiTemplate(ctx *base.SetupContext) map[string]interface{} {
	return AMI_TEMPLATE
}

func (app *Node) IsRemoteApp() bool {
	isRemote, _ := ami.IsRemoteApp(app.GetPath())
	return isRemote
}

func (app *Node) IsInstalled() bool {
	if isRemote, locator := ami.IsRemoteApp(app.GetPath()); isRemote {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			log.Warnf("Failed to check whether %s is installed on remote (%s)!", app.GetId(), err.Error())
			return false
		}
		defer session.Close()

		var output []byte
		output, _, err = session.IsRemoteAppInstalled(app.GetId())

		return err == nil && strings.Contains(string(output), "true")
	}
	return ami.IsAppInstalled(app.GetPath())
}

func (app *Node) SupportsRemote() bool {
	return true
}
