package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/ami"
)

var utilsCmd = &cobra.Command{
	Use:    "utils",
	Hidden: true,
}

var createRemoteCredentialsFileCmd = &cobra.Command{
	Use:   "create-remote-credentials",
	Short: "Create remote credentials file.",
	Long:  `Create remote credentials file.`,
	Run: func(cmd *cobra.Command, args []string) {
		directory, _ := cmd.Flags().GetString("path")
		username, _ := cmd.Flags().GetString("user")
		password, _ := cmd.Flags().GetString("pass")
		kind, _ := cmd.Flags().GetString("kind")

		switch ami.RemoteElevationKind(kind) {
		case ami.REMOTE_ELEVATION_SU:
		case ami.REMOTE_ELEVATION_SUDO:
		default:
			panic("Invalid kind of elevation.")
		}

		ami.WriteRemoteElevationCredentials(directory,
			&ami.RemoteConfiguration{
				Elevate: ami.RemoteElevationKind(kind),
			}, &ami.RemoteElevateCredentials{
				User:     username,
				Password: password,
				Kind:     ami.RemoteElevationKind(kind),
			})

	},
}

func init() {
	createRemoteCredentialsFileCmd.Flags().String("path", "", "Path to directory where the file will be created.")
	createRemoteCredentialsFileCmd.Flags().String("user", "", "Username for the remote server.")
	createRemoteCredentialsFileCmd.Flags().String("pass", "", "Password for the remote server.")
	createRemoteCredentialsFileCmd.Flags().String("kind", "sudo", "Kind of elevation (su or sudo).")

	utilsCmd.AddCommand(createRemoteCredentialsFileCmd)

	RootCmd.AddCommand(utilsCmd)
}
