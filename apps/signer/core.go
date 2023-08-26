package signer

import (
	"path"
	"strings"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
)

var (
	Id           string                 = constants.SignerAppId
	AMI_TEMPLATE map[string]interface{} = map[string]interface{}{
		"id":            constants.SignerAppId,
		"type":          map[string]interface{}{"id": "xtz.signer", "version": "latest"},
		"configuration": map[string]interface{}{},
		"user":          "",
	}
)

type Signer struct {
	Path string
}

// FromPath creates a new Signer instance with the specified path.
// The path parameter is the directory path to be associated with the Signer.
// If the path is empty, the default path will be used.
// It returns a pointer to the newly created Signer instance.
func FromPath(path string) *Signer {
	return &Signer{
		Path: path,
	}
}

func (app *Signer) GetAmiTemplate(ctx *base.SetupContext) map[string]interface{} {
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
	if app.Path != "" {
		return app.Path
	}
	return path.Join(cli.BBdir, Id)
}

func (app *Signer) GetId() string {
	return strings.ToLower(constants.SignerAppId)
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

func (app *Signer) IsRemoteApp() bool {
	return false
}
