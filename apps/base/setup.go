package base

import (
	"fmt"
	"path"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"
)

type SetupContext struct {
	Force bool
	// general info
	Configuration string
	Version       string
	User          string
	Branch        string

	// remote info
	Remote        string
	RemoteAuth    string
	RemoteElevate ami.RemoteElevationKind
	RemoteReset   bool

	Dal bool
}

func (ctx *SetupContext) ToRemoteConfiguration(app BakeBuddyApp) *ami.RemoteConfiguration {
	connectionDetails := system.GetRemoteConnectionDetails(ctx.Remote)
	localAppPath := app.GetLocalPath()

	return &ami.RemoteConfiguration{
		ElevationCredentialsDirectory: localAppPath,
		App:                           app.GetId(),
		Username:                      connectionDetails.Username,
		LocalUsername:                 ctx.User,
		Host:                          connectionDetails.Host,
		Port:                          connectionDetails.Port,
		InstancePath:                  constants.DefaultBBDirectory,
		Elevate:                       ctx.RemoteElevate,
		PrivateKey:                    path.Join(localAppPath, constants.PrivateKeyFile),
		PublicKey:                     path.Join(localAppPath, constants.PublicKeyFile),
	}
}

// promptElevation asks for the remote elevation password and the password that encrypts it on disk (empty stores it plain).
// It is a variable so tests can answer it without a terminal.
var promptElevation = func(appId string) (password string, encryptionPassword string) {
	password = util.RequirePasswordE(fmt.Sprintf("Enter password to use for elevation on %s remote:", appId), "Remote elevate requires password!", constants.ExitInternalError)
	encryptionPassword = util.RequirePasswordE("Enter password to encrypt credentials for elevation:", "failed to get password", constants.ExitInternalError)
	return password, encryptionPassword
}

// ConfigureRemoteLocator is shared by setup and unpack: it stores new elevation credentials when askElevation is set,
// writes the locator and its key pair (a new pair when rekey is set) to appDir, then prepares the remote host.
func ConfigureRemoteLocator(appDir string, locator *ami.RemoteConfiguration, rekey bool, askElevation bool, auth string) (*ami.RemoteConfiguration, error) {
	if askElevation && locator.Elevate != ami.REMOTE_ELEVATION_NONE {
		password, encryptionPassword := promptElevation(locator.App)
		credentials := &ami.RemoteElevateCredentials{Kind: locator.Elevate, Password: password}
		if err := ami.SaveRemoteElevationCredentials(appDir, locator, credentials, encryptionPassword); err != nil {
			return nil, fmt.Errorf("failed to save elevation credentials for %s: %w", locator.App, err)
		}
	}
	updated := ami.WriteRemoteLocator(appDir, locator, rekey)
	if err := ami.PrepareRemote(appDir, updated, auth); err != nil {
		return nil, fmt.Errorf("failed to prepare remote for %s: %w", locator.App, err)
	}
	return updated, nil
}
