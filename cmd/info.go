package cmd

import (
	"encoding/json"
	"fmt"

	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Prints runtime information about BB.",
	Long:  "Collects and prints runtime information about BB instance.",
	Run: func(cmd *cobra.Command, args []string) {
		timeout, _ := cmd.Flags().GetInt("timeout")
		if timeout <= 0 {
			timeout = 5
		}

		someModuleSelected := false
		installedModules := bb.GetInstalledModules()
		for _, v := range installedModules {
			infoOptions := v.GetAvailableInfoCollectionOptions()
			for _, option := range infoOptions {
				switch option.Type {
				case "bool":
					if checked, _ := cmd.Flags().GetBool(fmt.Sprintf("%s-%s", v.GetId(), option.Name)); checked {
						someModuleSelected = true
					}
				}
			}
			if someModuleSelected {
				break
			}
		}

		result := map[string]interface{}{}
		for _, v := range installedModules {
			shouldPrintModule, _ := cmd.Flags().GetBool(v.GetId())
			options := map[string]interface{}{
				"timeout": timeout,
			}
			for _, option := range v.GetAvailableInfoCollectionOptions() {
				switch option.Type {
				case "bool":
					if checked, _ := cmd.Flags().GetBool(fmt.Sprintf("%s-%s", v.GetId(), option.Name)); checked {
						options[option.Name] = true
					}
				}
			}
			if !someModuleSelected || shouldPrintModule {
				optionsJson, _ := json.Marshal(options)
				if cli.JsonLogFormat || cli.IsRemoteInstance {
					result[v.GetId()], _ = v.GetInfo(optionsJson)
				} else {
					err := v.PrintInfo(optionsJson)
					util.AssertE(err, fmt.Sprintf("Failed to collect %s's info!", v.GetId()))
				}
			}
		}

		if cli.JsonLogFormat || cli.IsRemoteInstance {
			output, err := json.Marshal(result)
			util.AssertEE(err, "Failed to serialize Bake Buddy runtime info!", cli.ExitSerializationFailed)
			fmt.Println(string(output))
			return
		}
	},
}

func init() {
	infoCmd.Flags().Int("timeout", 5, "How long to wait for collecting info.")
	for _, v := range bb.Modules {
		infoCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Prints info for %s.", v.GetId()))
		for _, option := range v.GetAvailableInfoCollectionOptions() {
			switch option.Type {
			case "bool":
				infoCmd.Flags().Bool(fmt.Sprintf("%s-%s", v.GetId(), option.Name), false, "")
			}
		}
	}
	RootCmd.AddCommand(infoCmd)
}
