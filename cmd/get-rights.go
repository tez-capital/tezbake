package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

type FailedToCollectRightsError struct{}

func (m *FailedToCollectRightsError) Error() string {
	return "Failed to collect baker rights!"
}

const (
	EndorsingRights string = "endorsing_rights"
	BakingRights    string = "baking_rights"
)

func getRightsForBlock(block int, delegate string, kind string, result chan []interface{}) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:8732/chains/main/blocks/head/helpers/%s?delegate=%s&max_priority=1&level=%d", kind, delegate, block))
	if err != nil {
		result <- []interface{}{
			map[string]interface{}{
				"status": "Failed to get baking rights!",
				"error":  err.Error(),
			},
		}
		return
	}

	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		result <- []interface{}{
			map[string]interface{}{
				"status": "Failed to get baking rights!",
				"error":  err.Error(),
			},
		}
		return
	}

	rights := make([]interface{}, 0)
	err = json.Unmarshal(bodyBytes, &rights)
	if err != nil {
		result <- []interface{}{
			map[string]interface{}{
				"status": "Failed to process baking rights!",
				"error":  err.Error(),
			},
		}
		return
	}
	result <- rights
}

func GetRights(futureBlocks int) (map[string]interface{}, error) {
	collectInfoOptions := map[string]interface{}{"timeout": 5}
	collectInfoOptionsJson, _ := json.Marshal(collectInfoOptions)
	nodeInfo, err := apps.Node.GetInfo(collectInfoOptionsJson)
	util.AssertE(err, "Failed to collect chain info from node!")
	keyHash, exitCode, err := apps.Signer.GetKeyHash("baker") // FIXME: technically it does not have to be called baker anymore
	util.AssertEE(err, "Failed to get baker key hash!", exitCode)

	if chainHead, ok := nodeInfo["chain_head"].(map[string]interface{}); ok {

		if headFloat, ok := chainHead["level"].(float64); ok {
			head := int(headFloat)
			endorsingBlockChans := make([]chan []interface{}, 0)
			bakingBlockChans := make([]chan []interface{}, 0)
			for i := 0; i < futureBlocks; i++ {
				endorsingBlockChan := make(chan []interface{})
				go getRightsForBlock(head+i+1, keyHash, EndorsingRights, endorsingBlockChan)
				endorsingBlockChans = append(endorsingBlockChans, endorsingBlockChan)

				bakingBlockChan := make(chan []interface{})
				go getRightsForBlock(head+i+1, keyHash, BakingRights, bakingBlockChan)
				bakingBlockChans = append(bakingBlockChans, bakingBlockChan)
			}
			endorsingRights := make([]interface{}, 0)
			bakingRights := make([]interface{}, 0)

			for _, ch := range endorsingBlockChans {
				endorsingRights = append(endorsingRights, <-ch...)
			}
			for _, ch := range bakingBlockChans {
				bakingRights = append(bakingRights, <-ch...)
			}
			return map[string]interface{}{
				"endorsing": endorsingRights,
				"baking":    bakingRights,
			}, nil
		}
	}
	return nil, &FailedToCollectRightsError{}
}

var getRightsCmd = &cobra.Command{
	Use:        "get-rights",
	Short:      "Gets baking and endorsing rights.",
	Long:       "Gets baking and endorsing rights of BB instance.",
	Deprecated: "This command is deprecated. Use mon peak app instead.",
	Run: func(cmd *cobra.Command, args []string) {
		futureBlocks, _ := cmd.Flags().GetInt("future-blocks")
		rights, err := GetRights(futureBlocks)
		util.AssertE(err, "Failed to get baker's rights!")

		result, err := json.Marshal(rights)
		util.AssertEE(err, "Failed to serialize baker's rights!", constants.ExitSerializationFailed)
		fmt.Println(string(result))
	},
}

func init() {
	getRightsCmd.Flags().IntP("future-blocks", "f", 60, "Sets how many future blocks should be probed for rights.")
	RootCmd.AddCommand(getRightsCmd)
}
