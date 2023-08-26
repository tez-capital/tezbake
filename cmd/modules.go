package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var modulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "Prints BB CLI modules.",
	Long:  "Prints BakeBuddy CLI modules.",
	Run: func(cmd *cobra.Command, args []string) {
		modulesTable := table.NewWriter()
		modulesTable.SetOutputMirror(os.Stdout)
		modulesTable.SetStyle(table.StyleLight)
		modulesTable.AppendHeader(table.Row{"Module", "Installed?"}, table.RowConfig{AutoMerge: true})

		result := map[string]interface{}{}
		for _, v := range bb.Modules {
			isInstalled := v.IsInstalled()
			result[v.GetId()] = map[string]interface{}{
				"installed": isInstalled,
			}
			modulesTable.AppendRow(table.Row{v.GetId(), isInstalled})
		}

		if cli.JsonLogFormat || cli.IsRemoteInstance {
			data, err := json.Marshal(result)
			util.AssertEE(err, "Failed to serialize modules info!", cli.ExitSerializationFailed)
			fmt.Println(string(data))
			return
		}

		modulesTable.Render()
	},
}

func init() {
	RootCmd.AddCommand(modulesCmd)
}
