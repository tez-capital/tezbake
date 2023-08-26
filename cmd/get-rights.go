package cmd

import (
	"encoding/json"
	"fmt"

	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	"github.com/spf13/cobra"
)

var getRightsCmd = &cobra.Command{
	Use:   "get-rights",
	Short: "Gets baking and endorsing rights.",
	Long:  "Gets baking and endorsing rights of BB instance.",
	Run: func(cmd *cobra.Command, args []string) {
		futureBlocks, _ := cmd.Flags().GetInt("future-blocks")
		rights, err := bb.GetRights(futureBlocks)
		util.AssertE(err, "Failed to get baker's rights!")

		result, err := json.Marshal(rights)
		util.AssertEE(err, "Failed to serialize baker's rights!", cli.ExitSerializationFailed)
		fmt.Println(string(result))
	},
}

func init() {
	getRightsCmd.Flags().IntP("future-blocks", "f", 60, "Sets how many future blocks should be probed for rights.")
	RootCmd.AddCommand(getRightsCmd)
}
