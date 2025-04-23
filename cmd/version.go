package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints BB CLI version.",
	Long:  "Prints BakeBuddy CLI version.",
	Run: func(cmd *cobra.Command, args []string) {
		shouldPrintAll, _ := cmd.Flags().GetBool("all")
		shouldPrintPackages, _ := cmd.Flags().GetBool("packages")
		shouldPrintBinaries, _ := cmd.Flags().GetBool("binaries")

		shouldPrintPackages = shouldPrintPackages || !shouldPrintBinaries || shouldPrintAll
		shouldPrintBinaries = shouldPrintBinaries || !shouldPrintPackages || shouldPrintAll

		appsToCollectFrom := GetAppsBySelectionCriteria(cmd, AppSelectionCriteria{
			InitialSelection:  InstalledApps,
			FallbackSelection: NoFallback,
		})

		if !shouldPrintAll && len(appsToCollectFrom) == 0 {
			if cli.JsonLogFormat {
				ver, _ := json.Marshal(constants.VERSION)
				fmt.Print(string(ver))
				return
			}
			fmt.Println(constants.VERSION)
			return
		}

		if shouldPrintAll && len(appsToCollectFrom) == 0 {
			appsToCollectFrom = apps.GetInstalledApps()
		}

		collectVersionOptions := &ami.CollectVersionsOptions{
			Packages: shouldPrintPackages,
			Binaries: shouldPrintBinaries,
		}

		versionTable := table.NewWriter()
		versionTable.SetOutputMirror(os.Stdout)
		versionTable.SetStyle(table.StyleLight)
		versionTable.AppendHeader(table.Row{"Tool", "Version"}, table.RowConfig{AutoMerge: true})
		versionTable.AppendRow(table.Row{"tezbake", constants.VERSION})

		result := map[string]interface{}{
			"cli": constants.VERSION,
		}
		for _, v := range appsToCollectFrom {
			versions, err := v.GetVersions(collectVersionOptions)
			util.AssertE(err, fmt.Sprintf("Failed to collect %s's versions!", v.GetId()))
			if v.GetId() == constants.NodeAppId && versions.IsRemote { // inject version from remote node
				versionTable.AppendRow(table.Row{"tezbake (remote)", versions.Cli})
				result["remote-cli"] = versions.Cli
			}
			result[v.GetId()] = versions
		}

		if cli.JsonLogFormat {
			verInfo, err := json.Marshal(result)
			util.AssertEE(err, "Failed to serialize version info!", constants.ExitSerializationFailed)
			fmt.Print(string(verInfo))
			return
		}

		for _, v := range appsToCollectFrom {
			versions := result[v.GetId()].(*ami.InstanceVersions)
			versionTable.AppendSeparator()
			versionTable.AppendRow(table.Row{v.GetLabel(), v.GetLabel()}, table.RowConfig{AutoMerge: true})
			versionTable.AppendSeparator()
			if shouldPrintPackages {
				versionTable.AppendRow(table.Row{"Package", "Version"}, table.RowConfig{AutoMerge: true})
				versionTable.AppendSeparator()
				for k, v := range versions.Packages {
					versionTable.AppendRow(table.Row{k, v})
				}
				versionTable.AppendSeparator()
			}
			if shouldPrintBinaries {
				versionTable.AppendRow(table.Row{"Binary", "Version"}, table.RowConfig{AutoMerge: true})
				versionTable.AppendSeparator()
				for k, v := range versions.Binaries {
					versionTable.AppendRow(table.Row{k, v})
				}
			}
		}

		versionTable.Render()
	},
}

func init() {
	versionCmd.Flags().BoolP("all", "a", false, "Prints version of all BB instance packages/binaries.")
	for _, v := range apps.All {
		versionCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Prints versions of %s.", v.GetId()))
	}
	versionCmd.Flags().Bool("packages", false, "Prints versions packages.")
	versionCmd.Flags().BoolP("binaries", "b", false, "Prints versions of binaries.")
	RootCmd.AddCommand(versionCmd)
}
