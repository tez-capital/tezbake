package cmd

import (
	"fmt"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	"github.com/spf13/cobra"
)

var voteCmd = &cobra.Command{
	Use:   "vote",
	Short: "Cast vote for proposal/s.",
	Long: `Submits vote for proposal/s based on provided voting period.
	
	Exploration:
	` + "`" + `bb-cli vote --period exploration <proposal> yay|nay|pass` + "`" + `
	Proposal:
	` + "`" + `bb-cli vote --period proposal <proposal1> <proposal2>` + "`" + `
	Promotion:
	` + "`" + `bb-cli vote --period promotion <proposal> yay|nay|pass` + "`" + `
	`,
	Run: func(cmd *cobra.Command, args []string) {
		period, _ := cmd.Flags().GetString("period")
		voteArgs := make([]string, 0)
		voteArgs = append(voteArgs, "client")
		// TODO:
		// if period == "auto" {
		// 	bb.Node.GetInfo()
		// }
		switch period {
		case "proposal":
			voteArgs = append(voteArgs, "submit", "proposals", "for", "baker")
			voteArgs = append(voteArgs, args...)

			_, err := ami.Execute(bb.Signer.GetPath(), voteArgs...)
			util.AssertE(err, "Failed to vote in '"+period+"' for "+fmt.Sprintf("%v", args)+"!")
		case "exploration":
			voteArgs = append(voteArgs, "submit", "ballot", "for", "baker")
			voteArgs = append(voteArgs, args...)

			_, err := ami.Execute(bb.Signer.GetPath(), voteArgs...)
			util.AssertE(err, "Failed to vote in '"+period+"' for "+fmt.Sprintf("%v", args)+"!")
		case "promotion":
			//tezos-client submit ballot for YOUR_ADDRESS Psithaca2MLRFYargivpo7YvUr7wUDqyxrdhC5CQq78mRvimz6A yay
			voteArgs = append(voteArgs, "submit", "ballot", "for", "baker")
			voteArgs = append(voteArgs, args...)

			_, err := ami.Execute(bb.Signer.GetPath(), voteArgs...)
			util.AssertE(err, "Failed to vote in '"+period+"' for "+fmt.Sprintf("%v", args)+"!")
		default:
			util.AssertBE(false, "Invalid period - '"+period+"'!", cli.ExitInvalidArgs)
		}
	},
}

func init() {
	voteCmd.Flags().String("period", "unknown", "Sets period to vote on.")
	voteCmd.Flags().SetInterspersed(false)
	RootCmd.AddCommand(voteCmd)
}
