package pay

import (
	"path"
	"strings"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
)

var (
	Id           string                 = constants.TezpayAppId
	AMI_TEMPLATE map[string]interface{} = map[string]interface{}{
		"id":            constants.TezpayAppId,
		"type":          map[string]interface{}{"id": "tzc.tezpay", "version": "latest"},
		"configuration": map[string]interface{}{},
		"user":          "",
	}
)

type Tezpay struct {
	Path string
}

// FromPath creates a new Node instance with the specified path.
// The path parameter is the directory path to be associated with the Node.
// If the path is empty, the default path will be used.
// It returns a pointer to the newly created Node instance.
func FromPath(path string) *Tezpay {
	return &Tezpay{
		Path: path,
	}
}

func (app *Tezpay) GetPath() string {
	if app.Path != "" {
		return app.Path
	}
	return path.Join(cli.BBdir, Id)
}

func (app *Tezpay) GetId() string {
	return strings.ToLower(constants.TezpayAppId)
}

func (app *Tezpay) GetLabel() string {
	return strings.ToUpper(app.GetId())
}

func (app *Tezpay) GetAmiTemplate(ctx *base.SetupContext) map[string]interface{} {
	return AMI_TEMPLATE
}
func (app *Tezpay) IsInstalled() bool {
	return ami.IsAppInstalled(app.GetPath())
}

func (app *Tezpay) SupportsRemote() bool {
	return false
}

func (app *Tezpay) IsRemoteApp() bool {
	return false
}
