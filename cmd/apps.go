package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Prints BB CLI apps.",
	Long:  "Prints BakeBuddy CLI apps.",
	Run: func(cmd *cobra.Command, args []string) {
		appsTable := table.NewWriter()
		appsTable.SetOutputMirror(os.Stdout)
		appsTable.SetStyle(table.StyleLight)
		appsTable.AppendHeader(table.Row{"App", "Installed?"}, table.RowConfig{AutoMerge: true})

		result := map[string]any{}
		for _, v := range apps.All {
			isInstalled := v.IsInstalled()
			result[v.GetId()] = map[string]any{
				"installed": isInstalled,
			}
			appsTable.AppendRow(table.Row{v.GetId(), isInstalled})
		}

		if cli.JsonLogFormat {
			data, err := json.Marshal(result)
			util.AssertEE(err, "Failed to serialize apps info!", constants.ExitSerializationFailed)
			fmt.Println(string(data))
			return
		}

		appsTable.Render()
	},
}

func init() {
	RootCmd.AddCommand(appsCmd)
}
