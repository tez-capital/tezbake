package cmd

import (
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

var executeAmiCmd = &cobra.Command{
	Use:    "execute-ami",
	Hidden: true,
	Short:  "executes command through tezbake",
	Run: func(cmd *cobra.Command, _ []string) {
		requiresElevation, _ := cmd.Flags().GetBool("elevate")
		if requiresElevation {
			system.RequireElevatedUser()
		}

		workingDir, _ := cmd.Flags().GetString("app")
		util.AssertBE(workingDir != "", "No app directory provided to execute.", constants.ExitInvalidArgs)
		jsonEncodedArgs, _ := cmd.Flags().GetString("args")
		base64EncodedArgs, _ := cmd.Flags().GetString("base64-args")

		var args []string
		if jsonEncodedArgs != "" {
			err := json.Unmarshal([]byte(jsonEncodedArgs), &args)
			util.AssertEE(err, "Failed to unmarshal JSON args: %v", constants.ExitInvalidArgs)
		}
		if base64EncodedArgs != "" {
			decodedBytes, err := base64.StdEncoding.DecodeString(base64EncodedArgs)
			util.AssertEE(err, "Failed to decode base64 string: %v", constants.ExitInvalidArgs)
			err = json.Unmarshal(decodedBytes, &args)
			util.AssertEE(err, "Failed to unmarshal JSON args: %v", constants.ExitInvalidArgs)
		}

		exitCode, err := ami.Execute(workingDir, args...)
		util.AssertEE(err, "Failed to execute ami command: %v", constants.ExitExternalError)

		os.Exit(exitCode)
	},
}

func init() {
	executeAmiCmd.Flags().Bool("elevate", false, "elevate before execution")
	executeAmiCmd.Flags().String("app", "", "Path to application directory")
	executeAmiCmd.Flags().String("args", "", "arguments to pass to the command")
	executeAmiCmd.Flags().String("base64-args", "", "basse64 encoded arguments to pass to the command")
	RootCmd.AddCommand(executeAmiCmd)
}
