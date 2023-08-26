package bb_module

import (
	"os"
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
	Remote         string
	RemoteAuth     string
	RemotePath     string
	RemoteUser     string // user to run bb under on remote
	RemoteElevate  string
	OneTimeElevate bool
	RemoteReset    bool
}

func (ctx *SetupContext) ToRemoteConfiguration(app IBakeBuddyModuleControl) *ami.RemoteConfiguration {
	connectionDetails := system.GetRemoteConnectionDetails(ctx.Remote)

	return &ami.RemoteConfiguration{
		App:          app.GetId(),
		Username:     connectionDetails.Username,
		Host:         connectionDetails.Host,
		Port:         connectionDetails.Port,
		InstancePath: ctx.RemotePath,
		Elevate:      ctx.RemoteElevate,
		PrivateKey:   path.Join(app.GetPath(), ami.PrivateKeyFile),
		PublicKey:    path.Join(app.GetPath(), ami.PublicKeyFile),
		ElevationCredentials: ami.RemoteElevateCredentials{
			REMOTE_SU_USER:   os.Getenv(ami.RemoteSuUser),
			REMOTE_SU_PASS:   os.Getenv(ami.RemoteSuPass),
			REMOTE_SUDO_PASS: os.Getenv(ami.RemoteSudoPass),
		},
	}
}
