package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/logging"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

var listLedgersCmd = &cobra.Command{
	Use:   "list-ledgers",
	Short: "Prints list of available ledgers.",
	Long:  "Collects and prits list of avaialble ledger ids.",
	Run: func(cmd *cobra.Command, args []string) {
		tezClientPath := path.Join(apps.Signer.GetPath(), "bin", "client")
		logging.Trace("Listing connected ledgers:", "tezClientPath", tezClientPath)
		output, err := exec.Command(tezClientPath, "list", "connected", "ledgers").CombinedOutput()
		if matched, _ := regexp.Match("Error:", output); err != nil || matched {
			fmt.Println(string(output))
			logging.Error("Failed to list ledgers!", "error", err)
			os.Exit(constants.ExitExternalError)
		}
		matchLedgers := regexp.MustCompile("## Ledger `(.*?)`")
		matches := matchLedgers.FindAllStringSubmatch(string(output), -1)
		if cli.JsonLogFormat {
			res := make([]string, 0)
			for _, v := range matches {
				if len(v) > 1 {
					res = append(res, v[1])
				}
			}
			output, err := json.Marshal(res)
			util.AssertEE(err, "Failed to serialize list of ledgers!", constants.ExitSerializationFailed)
			fmt.Println(string(output))
		} else {
			for _, v := range matches {
				if len(v) > 1 {
					fmt.Println(v[1])
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(listLedgersCmd)
}
