package cmd

import (
	"os"
	"path"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/archive"
	"alis.is/bb-cli/bb"
	bb_module_node "alis.is/bb-cli/bb/modules/node"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	//"github.com/pierrec/lz4"

	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

var bootstrapNodeCmd = &cobra.Command{
	Use:       "bootstrap-node [--tarball] <url> <block hash>",
	Short:     "Bootstraps Bake Buddy's Tezos node.",
	Long:      "Downloads bootstrap and imports it into node databse.",
	Args:      cobra.MinimumNArgs(0),
	ValidArgs: []string{"url", "block hash"},
	Run: func(cmd *cobra.Command, args []string) {
		shouldUseTarball, _ := cmd.Flags().GetBool("tarball")

		// TODO: move bootstrap routine to node module definition
		if !shouldUseTarball {
			bootstrapArgs := make([]string, 0)
			bootstrapArgs = append(bootstrapArgs, "bootstrap")
			bootstrapArgs = append(bootstrapArgs, args...)

			exitCode, err := ami.Execute(bb.Node.GetPath(), bootstrapArgs...)
			util.AssertEE(err, "Failed to bootstrap tezos node", exitCode)

			log.Info("Upgrading storage...")
			exitCode, err = bb_module_node.Module.UpgradeStorage()
			util.AssertEE(err, "Failed to upgrade tezos storage", exitCode)

			os.Exit(exitCode)
		}

		if isRemote, locator := ami.IsRemoteApp(bb.Node.GetPath()); isRemote {
			session, err := locator.OpenAppRemoteSessionS()
			util.AssertE(err, "Failed to open remote session!")
			exitCode, _ := session.ProxyToRemoteApp()
			os.Exit(exitCode)
		}

		system.RequireElevatedUser()
		// curl -LfsS "https://mainnet.xtz-shots.io/rolling-tarball" | lz4 -d | tar -x -C "/var/tezos"
		log.Trace("Preparing directories...")
		tmpDir := path.Join(bb.Node.GetPath(), ".tarball-tmp")
		util.AssertEE(os.MkdirAll(tmpDir, 0700), "Failed to create directory structure for tarball download!", cli.ExitIOError)
		newDataDir := path.Join(tmpDir, "new-data")
		newDataContentDir := path.Join(newDataDir, "node", "data")
		util.AssertEE(os.MkdirAll(newDataDir, 0700), "Failed to create directory structure for tarball download!", cli.ExitIOError)

		tarballFileName := "bootstrap.tarball"
		tarballFilePath := path.Join(tmpDir, tarballFileName)

		tarballUrl := "https://mainnet.xtz-shots.io/rolling-tarball"
		if len(args) > 0 {
			tarballUrl = args[0]
		}
		log.Info("Downloading bootstrap tarball...")
		os.Remove(tarballFilePath)
		util.AssertE(util.DownloadFile(tarballUrl, tarballFilePath, true), "Failed to download tarball!")

		log.Info("Extracting the tarball...")
		inputFile, err := os.Open(tarballFilePath)
		util.AssertEE(err, "Failed to open downloaded tarball!", cli.ExitIOError)
		defer inputFile.Close()

		util.AssertE(archive.UntarLz4(inputFile, newDataDir), "Failed to extract downloaded tarball!")

		wasRunning, _ := bb.Node.IsServiceStatus(ami.NodeService, "running")
		if wasRunning {
			log.Info("Node services up. Running bootstrap in live mode...")
			bb.Node.Stop()
		}
		nodeDataDir := path.Join(bb.Node.GetPath(), "data", ".tezos-node")
		util.AssertEE(os.MkdirAll(path.Dir(nodeDataDir), 0700), "Failed to create directory tree for node data!", cli.ExitIOError)
		identityFileName := "identity.json"
		peersFileName := "peers.json"
		configFileName := "config.json"

		os.Rename(path.Join(nodeDataDir, identityFileName), path.Join(newDataContentDir, identityFileName))
		os.Rename(path.Join(nodeDataDir, peersFileName), path.Join(newDataContentDir, peersFileName))
		os.Rename(path.Join(nodeDataDir, configFileName), path.Join(newDataContentDir, configFileName))

		err = os.RemoveAll(nodeDataDir)
		util.AssertEE(err, "Failed to remove old node data directory!", cli.ExitIOError)
		err = os.Rename(newDataContentDir, nodeDataDir)
		util.AssertEE(err, "Failed to deploy new data directory!", cli.ExitIOError)

		nodeDef, _, err := bb.Node.LoadAppDefinition()
		util.AssertEE(err, "Failed to load node definition!", cli.ExitAppConfigurationLoadFailed)
		nodeUser, ok := nodeDef["user"].(string)
		util.AssertBE(ok, "Failed to get username from node!", cli.ExitInvalidUser)

		log.Info("Upgrading storage...")
		exitCode, err := bb_module_node.Module.UpgradeStorage()
		util.AssertEE(err, "Failed to upgrade tezos storage", exitCode)

		util.ChownR(nodeUser, path.Join(bb.Node.GetPath(), "data"))

		os.RemoveAll(tmpDir)

		if wasRunning {
			bb.Node.Start()
		}
		log.Info("Node bootstrapped from tarball successfully...")
	},
}

func init() {
	bootstrapNodeCmd.Flags().Bool("tarball", false, "EXPERIMENTAL - Bootstraps node from tarball. (supports live migration)")
	//bootstrapNodeCmd.Flags().String("tarball-url", "https://mainnet.xtz-shots.io/rolling-tarball", "Bootstraps node from tarball. (supports life migration)")
	RootCmd.AddCommand(bootstrapNodeCmd)
}
