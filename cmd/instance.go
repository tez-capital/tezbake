package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
)

var instanceCmd = &cobra.Command{
	Use:                "instance [alias] [command]",
	Short:              "Executes command on a specific tezbake instance",
	Long:               "Proxies the command to the specified tezbake instance by setting the appropriate path.",
	Aliases:            []string{"i"},
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Manually handle help since DisableFlagParsing is true
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				return cmd.Help()
			}
		}

		if len(args) < 1 {
			return fmt.Errorf("instance alias is required")
		}

		if args[0] == "list" {
			return listCmd.RunE(cmd, args)
		}

		alias := args[0]
		// Determine the new path for the instance
		instancePath := filepath.Join(constants.DefaultBBDirectory, "instances", alias)

		// Prepare arguments for the recursive execution
		// We prepend --path <instancePath> to the arguments
		newArgs := append([]string{"--path", instancePath}, args[1:]...)

		// Reset the RootCmd args and execute it again with the new arguments
		RootCmd.SetArgs(newArgs)
		return RootCmd.Execute()
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available instances",
	Long:  "Lists all available tezbake instances found in the default directory.",
	RunE: func(cmd *cobra.Command, args []string) error {
		bbDir := cli.BBdir
		if bbDir == "" {
			bbDir = constants.DefaultBBDirectory
		}

		instances := []string{}

		// Check for default instance
		// Default instance exists if there is any other directory in the /bake-buddy except instances
		entries, err := os.ReadDir(bbDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && entry.Name() != "instances" {
					instances = append(instances, "default")
					break
				}
			}
		}

		// Check for other instances
		instancesDir := filepath.Join(bbDir, "instances")
		entries, err = os.ReadDir(instancesDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					instances = append(instances, entry.Name())
				}
			}
		}

		if cli.JsonLogFormat {
			output := make([]map[string]string, 0, len(instances))
			for _, instance := range instances {
				location := filepath.Join(bbDir, "instances", instance)
				if instance == "default" {
					location = bbDir
				}
				output = append(output, map[string]string{"name": instance, "location": location})
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
		}

		for _, instance := range instances {
			location := filepath.Join(bbDir, "instances", instance)
			if instance == "default" {
				location = bbDir
			}
			fmt.Fprintf(cmd.OutOrStdout(), "- %s\t\t%s\n", instance, location)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(instanceCmd)
	instanceCmd.AddCommand(listCmd)
}
