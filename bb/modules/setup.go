package bb_module

import (
	"path"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/system"
)

type SetupContext struct {
	Force bool
	// general info
	Configuration string
	Version       string
	User          string
	Branch        string

	// remote info
	Remote                string
	RemoteAuth            string
	RemotePath            string
	RemoteUser            string // user to run bb under on remote
	RemoteElevate         ami.ERemoteElevationKind
	RemoteElevateUser     string // user to elevate to on remote
	RemoteElevatePassword string
	OneTimeElevate        bool
	RemoteReset           bool
}

func (ctx *SetupContext) ToRemoteConfiguration(app IBakeBuddyModuleControl) *ami.RemoteConfiguration {
	connectionDetails := system.GetRemoteConnectionDetails(ctx.Remote)

	return &ami.RemoteConfiguration{
		ElevationCredentialsDirectory: app.GetPath(),
		App:                           app.GetId(),
		Username:                      connectionDetails.Username,
		Host:                          connectionDetails.Host,
		Port:                          connectionDetails.Port,
		InstancePath:                  ctx.RemotePath,
		Elevate:                       ctx.RemoteElevate,
		PrivateKey:                    path.Join(app.GetPath(), ami.PrivateKeyFile),
		PublicKey:                     path.Join(app.GetPath(), ami.PublicKeyFile),
	}
}

func (ctx *SetupContext) ToRemoteElevateCredentials() *ami.RemoteElevateCredentials {
	if ctx.RemoteElevate == ami.REMOTE_ELEVATION_NONE {
		return nil
	}
	return &ami.RemoteElevateCredentials{
		Kind:     ami.ERemoteElevationKind(ctx.RemoteElevate),
		User:     ctx.RemoteElevateUser,
		Password: ctx.RemoteElevatePassword,
	}
}
