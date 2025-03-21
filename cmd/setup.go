package cmd

import (
	"fmt"
	"os"
	"os/user"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	"github.com/AlecAivazis/survey/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	RemoteNode          = "remote-node"
	RemoteAuth          = "remote-auth"
	RemoteElevate       = "remote-elevate"
	RemoteElevateUser   = "remote-elevate-user"
	RemoteUser          = "remote-user"
	RemotePath          = "remote-path"
	RemoteReset         = "remote-reset"
	UpgradeStorage      = "upgrade-storage"
	Branch              = "branch"
	Id                  = "id"
	User                = "user"
	Node                = "node"
	NodeVersion         = "node-version"
	NodeConfiguration   = "node-configuration"
	Signer              = "signer"
	SignerVersion       = "signer-version"
	SignerConfiguration = "signer-configuration"
	SetupAmi            = "setup-ami"
	Force               = "force"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setups BB.",
	Long:  "Installs and configures BB instance.",
	Run: func(cmd *cobra.Command, args []string) {
		username := util.GetCommandStringFlag(cmd, User)
		if !system.IsElevated() {
			system.RequireElevatedUser("--user=" + username)
		}

		util.AssertBE(username != "", "User not specified", constants.ExitInvalidUser)
		if username == "root" && !cli.IsRemoteInstance {
			proceed := false
			if system.IsTty() {
				prompt := &survey.Confirm{
					Message: "You are going to setup tezbake as root. This is not recommended. Do you want to proceed anyway?",
				}
				survey.AskOne(prompt, &proceed)
			}
			if !proceed {
				os.Exit(constants.ExitOperationCanceled)
			}
		}

		id := util.GetCommandStringFlagS(cmd, Id)
		force := util.GetCommandBoolFlagS(cmd, Force)
		util.AssertBE(id != "", "Id not specified", constants.ExitInvalidId)
		util.AssertBE(id != "bb-default" || cli.BBdir == constants.DefaultBBDirectory, "Please specify id for baker. 'default' id is allowed only for bake buddy installed in '"+constants.DefaultBBDirectory+"' path!", constants.ExitInvalidId)
		cli.BBInstanceId = id

		if util.GetCommandBoolFlagS(cmd, SetupAmi) || cli.IsRemoteInstance {
			// install ami by default in case of remote instance
			exitCode, err := ami.Install()
			util.AssertEE(err, "Failed to install ami and eli!", exitCode)
		}

		for _, v := range GetAppsBySelectionCriteria(cmd, AppSelectionCriteria{
			InitialSelection:  AllApps,
			FallbackSelection: ImplicitApps,
		}) {
			appId := v.GetId()
			if cli.IsRemoteInstance && !v.SupportsRemote() {
				log.Debug(fmt.Sprintf("'%s' does not support remote. Skipping...", appId))
				continue
			}

			branch := util.GetCommandStringFlagS(cmd, fmt.Sprintf("%s-branch", appId))
			if branch == "" {
				branch = util.GetCommandStringFlagS(cmd, Branch)
			}

			ctx := &apps.SetupContext{
				Configuration: util.GetCommandStringFlagS(cmd, fmt.Sprintf("%s-configuration", appId)),
				Version:       util.GetCommandStringFlagS(cmd, fmt.Sprintf("%s-version", appId)),
				Branch:        branch,
				User:          username,

				Remote:      util.GetCommandStringFlagS(cmd, RemoteNode),
				RemoteAuth:  util.GetCommandStringFlagS(cmd, RemoteAuth),
				RemotePath:  util.GetCommandStringFlagS(cmd, RemotePath),
				RemoteUser:  util.GetCommandStringFlagS(cmd, RemoteUser),
				RemoteReset: util.GetCommandBoolFlagS(cmd, RemoteReset),

				RemoteElevate:     ami.RemoteElevationKind(util.GetCommandStringFlagS(cmd, RemoteElevate)),
				RemoteElevateUser: util.GetCommandStringFlagS(cmd, RemoteElevateUser),
				//RemoteElevatePassword: remoteElevatePassword,

				Force: force,
			}

			if v.IsInstalled() && !force && !cli.IsRemoteInstance {
				proceed := false
				if system.IsTty() {
					prompt := &survey.Confirm{
						Message: fmt.Sprintf("Existing setup of '%s' found. Do you want to %s?", v.GetId(), v.GetSetupKind()),
					}
					survey.AskOne(prompt, &proceed)
				}
				if !proceed {
					os.Exit(constants.ExitOperationCanceled)
				}
			}

			exitCode, err := v.Setup(ctx)
			util.AssertEE(err, fmt.Sprintf("Failed to setup '%s'!", v.GetId()), exitCode)
		}

		util.ChownR(username, cli.BBdir)
		log.Info("Setup successful")
	},
}

func init() {
	setupCmd.Flags().BoolP(SetupAmi, "a", false, "Install latest ami during the BB setup.")
	setupCmd.Flags().Bool(Force, false, "Force setup - potentially overwriting existing installation.")

	user, err := user.Current()
	if err != nil {
		log.Warn("Failed to get current user!")
		setupCmd.Flags().StringP(User, "u", "", "User you want to operate BB under.")
	} else {
		setupCmd.Flags().StringP(User, "u", user.Username, "User you want to operate BB under.")
	}

	for _, v := range apps.All {
		setupCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Setups %s.", v.GetId()))
		setupCmd.Flags().String(fmt.Sprintf("%s-configuration", v.GetId()), "{}", fmt.Sprintf("Sets %s configuration.", v.GetId()))
		setupCmd.Flags().String(fmt.Sprintf("%s-version", v.GetId()), "latest", fmt.Sprintf("Sets %s configuration.", v.GetId()))
		setupCmd.Flags().String(fmt.Sprintf("%s-branch", v.GetId()), "", fmt.Sprintf("Sets %s configuration.", v.GetId()))
	}

	setupCmd.Flags().StringP(Id, "i", "bb-default", "Id of BB instance.")

	setupCmd.Flags().String(RemoteNode, "", "username:<ssh key file>@address (experimental)")
	setupCmd.Flags().String(RemoteAuth, "", "pass|key:<path to key>  (experimental)")
	setupCmd.Flags().String(RemotePath, constants.DefaultRemoteBBDirectory, "where on remote install node - defaults to '/bake-buddy/node' (experimental)")
	setupCmd.Flags().String(RemoteElevate, "", "only 'sudo' supported now (experimental)")
	setupCmd.Flags().String(RemoteElevateUser, "", "user to elevate to (experimental)")
	setupCmd.Flags().MarkHidden(RemoteElevateUser)
	setupCmd.Flags().String(RemoteUser, constants.DefaultRemoteUser, "Sets user remote node will be operated under. (experimental, ignored if remote uses elevation)")

	setupCmd.Flags().Bool(RemoteReset, false, "Resets and reconfigures remote node locator. (experimental)")

	setupCmd.Flags().String(Branch, "main", "Select package release branch you want to setup (addets node and signer app only).")
	RootCmd.AddCommand(setupCmd)
}
