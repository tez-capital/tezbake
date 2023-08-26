package bb_module_signer

import (
	"path"
	"strings"

	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/system"
)

var (
	Id           string                 = ami.Signer
	AMI_TEMPLATE map[string]interface{} = map[string]interface{}{
		"id":            ami.Signer,
		"type":          map[string]interface{}{"id": "xtz.signer", "version": "latest"},
		"configuration": map[string]interface{}{},
		"user":          "",
	}
	Module = &Signer{}
)

type Signer struct {
}

func (app *Signer) GetAmiTemplate(ctx *bb_module.SetupContext) map[string]interface{} {
	if ctx.Remote != "" {
		connectionsDetails := system.GetRemoteConnectionDetails(ctx.Remote)
		// from xtz.signer
		// REMOTE_SSH_PORT = am.app.get_configuration("REMOTE_SSH_PORT", "22"),
		// REMOTE_SSH_KEY = am.app.get_configuration("REMOTE_SSH_KEY"),
		// REMOTE_NODE = am.app.get_configuration("REMOTE_NODE"),
		configuration := AMI_TEMPLATE["configuration"].(map[string]interface{})
		configuration["REMOTE_NODE"] = connectionsDetails.Username + "@" + connectionsDetails.Host
		configuration["REMOTE_SSH_PORT"] = connectionsDetails.Port
		configuration["REMOTE_SSH_KEY"] = path.Join(path.Dir(app.GetPath()), "node", "idkey")
	}
	return AMI_TEMPLATE
}

func (app *Signer) GetPath() string {
	return path.Join(cli.BBdir, Id)
}

func (app *Signer) GetId() string {
	return strings.ToLower(ami.Signer)
}

func (app *Signer) GetLabel() string {
	return strings.ToUpper(app.GetId())
}

func (app *Signer) IsInstalled() bool {
	return ami.IsAppInstalled(app.GetPath())
}

func (app *Signer) SupportsRemote() bool {
	return false
}
