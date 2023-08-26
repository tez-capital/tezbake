package cmd

import (
	"encoding/base64"
	"log"
	"os"
	"os/exec"

	"github.com/tez-capital/tezbake/system"

	shellquote "github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

var executeCmd = &cobra.Command{
	Use:    "execute",
	Hidden: true,
	Short:  "executes command through tezbake",
	Run: func(cmd *cobra.Command, args []string) {
		requiresElevation, _ := cmd.Flags().GetBool("elevate")
		if requiresElevation && !system.IsElevated() {
			system.RequireElevatedUser()
		}

		commandStr, _ := cmd.Flags().GetString("command")
		base64Str, _ := cmd.Flags().GetString("base64")

		if base64Str != "" {
			decodedBytes, err := base64.StdEncoding.DecodeString(base64Str)
			if err != nil {
				log.Fatalf("Failed to decode base64 string: %v", err)
			}
			commandStr = string(decodedBytes)
		}

		if commandStr == "" {
			log.Fatal("No command provided to execute.")
		}
		commandsParts, err := shellquote.Split(commandStr)
		if err != nil {
			log.Fatalf("Failed to parse command: %v", err)
		}
		if len(commandsParts) == 0 {
			log.Fatal("No command provided to execute.")
		}
		name := commandsParts[0]
		arg := commandsParts[1:]

		c := exec.Command(name, arg...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		_ = c.Run()
		os.Exit(c.ProcessState.ExitCode())
	},
}

func init() {
	executeCmd.Flags().Bool("elevate", false, "elevate before execution")
	executeCmd.Flags().String("command", "", "command to execute")
	executeCmd.Flags().String("base64", "", "command encoded in base64 to execute")

	RootCmd.AddCommand(executeCmd)
}
