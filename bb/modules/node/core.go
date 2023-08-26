package bb_module_node

import (
	"fmt"
	"path"
	"strings"

	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"
	"alis.is/bb-cli/cli"

	log "github.com/sirupsen/logrus"
)

var (
	Id           string                 = ami.Node
	AMI_TEMPLATE map[string]interface{} = map[string]interface{}{
		"id":   ami.Node,
		"type": map[string]interface{}{"id": "xtz.node", "version": "latest"},
		"configuration": map[string]interface{}{
			"NODE_TYPE": "baker",
		},
		"user": "",
	}
	Module = &Node{}
)

type Node struct {
}

func (app *Node) GetPath() string {
	return path.Join(cli.BBdir, Id)
}

func (app *Node) GetId() string {
	return strings.ToLower(ami.Node)
}

func (app *Node) GetLabel() string {
	if isRemote, locator := ami.IsRemoteApp(app.GetPath()); isRemote {
		return strings.ToUpper(fmt.Sprintf("%s (REMOTE - %s:%s)", app.GetId(), locator.Host, locator.Port))
	}
	return strings.ToUpper(app.GetId())
}

func (app *Node) GetAmiTemplate(ctx *bb_module.SetupContext) map[string]interface{} {
	return AMI_TEMPLATE
}

func (app *Node) IsInstalled() bool {
	if isRemote, locator := ami.IsRemoteApp(app.GetPath()); isRemote {
		session, err := locator.OpenAppRemoteSessionS()
		if err == nil {
			output, _, err := session.IsRemoteModuleInstalled(app.GetId())
			if err == nil {
				return strings.Contains(string(output), "true")
			}
		}
		log.Warnf("Failed to check whether %s is installed on remote (%s)!", app.GetId(), err.Error())
		return false
	}
	return ami.IsAppInstalled(app.GetPath())
}

func (app *Node) SupportsRemote() bool {
	return true
}
