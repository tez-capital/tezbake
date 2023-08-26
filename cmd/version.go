package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

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

		someModuleSelected := false
		installedModules := bb.GetInstalledModules()
		for _, v := range installedModules {
			if checked, _ := cmd.Flags().GetBool(v.GetId()); checked {
				someModuleSelected = true
			}
			if someModuleSelected {
				break
			}
		}

		if !shouldPrintAll && !someModuleSelected {
			if cli.JsonLogFormat {
				ver, _ := json.Marshal(bb.VERSION)
				fmt.Print(string(ver))
				return
			}
			fmt.Println(bb.VERSION)
			return
		}

		collectVersionOptions := &ami.CollectVersionsOptions{
			Packages: shouldPrintAll || shouldPrintPackages,
			Binaries: shouldPrintAll || shouldPrintBinaries,
		}

		versionTable := table.NewWriter()
		versionTable.SetOutputMirror(os.Stdout)
		versionTable.SetStyle(table.StyleLight)
		versionTable.AppendHeader(table.Row{"Tool", "Version"}, table.RowConfig{AutoMerge: true})
		versionTable.AppendRow(table.Row{"bb-cli", bb.VERSION})

		result := map[string]interface{}{
			"cli": bb.VERSION,
		}
		for _, v := range installedModules {
			if !someModuleSelected || util.GetCommandBoolFlagS(cmd, v.GetId()) {
				versions, err := v.GetVersions(collectVersionOptions)
				util.AssertE(err, fmt.Sprintf("Failed to collect %s's versions!", v.GetId()))
				if v.GetId() == ami.Node && versions.IsRemote { // inject version from remote node
					versionTable.AppendRow(table.Row{"bb-cli (remote)", versions.Cli})
					result["remote-cli"] = versions.Cli
				}
				result[v.GetId()] = versions
			}
		}

		if cli.JsonLogFormat || cli.IsRemoteInstance {
			verInfo, err := json.Marshal(result)
			util.AssertEE(err, "Failed to serialize version info!", cli.ExitSerializationFailed)
			fmt.Print(string(verInfo))
			return
		}

		for _, v := range installedModules {
			if someModuleSelected && !util.GetCommandBoolFlagS(cmd, v.GetId()) {
				continue
			}
			versions := result[v.GetId()].(*ami.InstanceVersions)
			versionTable.AppendSeparator()
			versionTable.AppendRow(table.Row{v.GetLabel(), v.GetLabel()}, table.RowConfig{AutoMerge: true})
			versionTable.AppendSeparator()
			versionTable.AppendRow(table.Row{"Package", "Version"}, table.RowConfig{AutoMerge: true})
			versionTable.AppendSeparator()
			for k, v := range versions.Packages {
				versionTable.AppendRow(table.Row{k, v})
			}
			versionTable.AppendSeparator()
			versionTable.AppendRow(table.Row{"Binary", "Version"}, table.RowConfig{AutoMerge: true})
			versionTable.AppendSeparator()
			for k, v := range versions.Binaries {
				versionTable.AppendRow(table.Row{k, v})
			}
		}

		versionTable.Render()
	},
}

func init() {
	versionCmd.Flags().BoolP("all", "a", false, "Prints version of all BB instance packages/binaries.")
	for _, v := range bb.Modules {
		versionCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Prints versions of %s.", v.GetId()))
	}
	versionCmd.Flags().Bool("packages", false, "Prints versions packages.")
	versionCmd.Flags().BoolP("binaries", "b", false, "Prints versions of binaries.")
	RootCmd.AddCommand(versionCmd)
}
