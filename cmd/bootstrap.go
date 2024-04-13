package cmd

import (
	"os"
	"time"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/util"

	//"github.com/pierrec/lz4"

	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

type ArtifactKind string

const (
	TezosSnapshot ArtifactKind = "tezos-snapshot"
)

type snapshot struct {
	BlockHeight    int64        `json:"block_height"`
	BlockHash      string       `json:"block_hash"`
	BlockTimestamp time.Time    `json:"block_timestamp"`
	ChainName      string       `json:"chain_name"`
	HistoryMode    string       `json:"history_mode"`
	Url            string       `json:"url"`
	ArtifactKind   ArtifactKind `json:"artifact_type"`
}

type snapshots struct {
	Data []snapshot `json:"data"`
}

var bootstrapNodeCmd = &cobra.Command{
	Use:       "bootstrap-node [--no-check] <url> <block hash>",
	Short:     "Bootstraps Bake Buddy's Tezos node.",
	Long:      "Downloads bootstrap and imports it into node databse.",
	Args:      cobra.MinimumNArgs(0),
	ValidArgs: []string{"url", "block hash"},
	Run: func(cmd *cobra.Command, args []string) {
		disableSnapshotCheck, _ := cmd.Flags().GetBool("no-check")

		var chosenSnapshot *snapshot = nil
		//artifactKind := TezosSnapshot

		// if system.IsTty() && len(args) == 0 {
		// 	resp, err := http.Get("https://xtz-shots.io/tezos-snapshots.json")
		// 	util.AssertEE(err, "Failed to get list of available snapshots", constants.ExitExternalError)
		// 	defer resp.Body.Close()

		// 	body, err := io.ReadAll(resp.Body)
		// 	util.AssertEE(err, "Failed to get list of available snapshots", constants.ExitExternalError)

		// 	var snapshots snapshots
		// 	err = json.Unmarshal(body, &snapshots)
		// 	util.AssertEE(err, "Failed to get list of available snapshots", constants.ExitExternalError)

		// 	var (
		// 		chainName   string
		// 		historyMode string
		// 	)
		// 	prompt := &survey.Select{
		// 		Message: "Which chain do you want to bootstrap for?",
		// 		Options: []string{"mainnet", "ghostnet"},
		// 	}
		// 	err = survey.AskOne(prompt, &chainName)
		// 	util.AssertEE(err, "failed to get user input", constants.ExitUserInvalidInput)

		// 	if !disableSnapshotCheck { // if parameter wasn't set, ask user
		// 		prompt = &survey.Select{
		// 			Message: "What kind of artifact do you want use to bootstrap?",
		// 			Options: []string{string(Tarball), string(TezosSnapshot)},
		// 		}
		// 		err = survey.AskOne(prompt, &artifactKind)
		// 		util.AssertEE(err, "failed to get user input", constants.ExitUserInvalidInput)
		// 	}

		// 	prompt = &survey.Select{
		// 		Message: "What kind of history mode should be used?",
		// 		Options: []string{"rolling", "archive"},
		// 	}
		// 	err = survey.AskOne(prompt, &historyMode)
		// 	util.AssertEE(err, "failed to get user input", constants.ExitUserInvalidInput)

		// 	usableSnapshots := lo.Filter(snapshots.Data, func(s snapshot, _ int) bool {
		// 		return s.ArtifactKind == artifactKind && s.ChainName == chainName && s.HistoryMode == historyMode
		// 	})
		// 	if len(usableSnapshots) == 0 {
		// 		log.Error("No snapshots found for given criteria!")
		// 		os.Exit(constants.ExitExternalError)
		// 	}

		// 	slices.SortFunc(usableSnapshots, func(i, j snapshot) int {
		// 		return int(j.BlockHeight - i.BlockHeight)
		// 	})
		// 	snapshotOptions := lo.Map(usableSnapshots, func(s snapshot, _ int) string {
		// 		return fmt.Sprintf("%d - %s", s.BlockHeight, s.BlockTimestamp.Format("2006-01-02 15:04:05"))
		// 	})

		// 	prompt = &survey.Select{
		// 		Message: "Please choose snapshot you want to use:",
		// 		Options: snapshotOptions,
		// 	}
		// 	var snapshotBlockAndTime string
		// 	err = survey.AskOne(prompt, &snapshotBlockAndTime)
		// 	util.AssertEE(err, "failed to get user input", constants.ExitUserInvalidInput)

		// 	snapshotBlock, _ := strconv.ParseInt(snapshotBlockAndTime[:strings.Index(snapshotBlockAndTime, " - ")], 10, 64)
		// 	snapshot, found := lo.Find(usableSnapshots, func(s snapshot) bool {
		// 		return s.BlockHeight == snapshotBlock
		// 	})
		// 	util.AssertBE(found, "Failed to find snapshot!", constants.ExitInternalError)
		// 	chosenSnapshot = &snapshot
		// }

		//if artifactKind == TezosSnapshot {
		bootstrapArgs := make([]string, 0)
		bootstrapArgs = append(bootstrapArgs, "bootstrap")

		if chosenSnapshot != nil {
			bootstrapArgs = append(bootstrapArgs, chosenSnapshot.Url)
			bootstrapArgs = append(bootstrapArgs, chosenSnapshot.BlockHash)
		} else {
			bootstrapArgs = append(bootstrapArgs, args...)
		}
		if disableSnapshotCheck {
			bootstrapArgs = append(bootstrapArgs, "--no-check")
		}
		exitCode, err := apps.Node.Execute(bootstrapArgs...)
		util.AssertEE(err, "Failed to bootstrap tezos node", exitCode)

		log.Info("Upgrading storage...")
		exitCode, err = apps.Node.UpgradeStorage()
		util.AssertEE(err, "Failed to upgrade tezos storage", exitCode)

		os.Exit(exitCode)
		//}

		// if isRemote, locator := ami.IsRemoteApp(bb.Node.GetPath()); isRemote {
		// 	session, err := locator.OpenAppRemoteSessionS()
		// 	util.AssertE(err, "Failed to open remote session!")
		// 	exitCode, _ := session.ProxyToRemoteApp()
		// 	os.Exit(exitCode)
		// }

		// // tarball bootstrap
		// system.RequireElevatedUser()
		// // curl -LfsS "https://mainnet.xtz-shots.io/rolling-tarball" | lz4 -d | tar -x -C "/var/tezos"
		// log.Trace("Preparing directories...")
		// tmpDir := path.Join(bb.Node.GetPath(), ".tarball-tmp")
		// util.AssertEE(os.MkdirAll(tmpDir, 0700), "Failed to create directory structure for tarball download!", constants.ExitIOError)
		// newDataDir := path.Join(tmpDir, "new-data")
		// newDataContentDir := path.Join(newDataDir, "node", "data")
		// util.AssertEE(os.MkdirAll(newDataDir, 0700), "Failed to create directory structure for tarball download!", constants.ExitIOError)

		// tarballFileName := "bootstrap.tarball"
		// tarballFilePath := path.Join(tmpDir, tarballFileName)

		// util.AssertBE(chosenSnapshot != nil || len(args) > 0, "No snapshot url provided!", constants.ExitInvalidArgs)
		// var tarbalUrl string
		// if chosenSnapshot != nil && len(args) == 0 {
		// 	tarbalUrl = chosenSnapshot.Url
		// } else {
		// 	tarbalUrl = args[0]
		// }

		// log.Info("Downloading bootstrap tarball...")
		// os.Remove(tarballFilePath)
		// util.AssertE(util.DownloadFile(tarbalUrl, tarballFilePath, true), "Failed to download tarball!")

		// log.Info("Extracting the tarball...")
		// inputFile, err := os.Open(tarballFilePath)
		// util.AssertEE(err, "Failed to open downloaded tarball!", constants.ExitIOError)
		// defer inputFile.Close()

		// util.AssertE(archive.UntarLz4(inputFile, newDataDir), "Failed to extract downloaded tarball!")

		// wasRunning, _ := bb.Node.IsServiceStatus(ami.NodeService, "running")
		// if wasRunning {
		// 	log.Info("Node services up. Running bootstrap in live mode...")
		// 	bb.Node.Stop()
		// }
		// nodeDataDir := path.Join(bb.Node.GetPath(), "data", ".tezos-node")
		// util.AssertEE(os.MkdirAll(path.Dir(nodeDataDir), 0700), "Failed to create directory tree for node data!", constants.ExitIOError)
		// identityFileName := "identity.json"
		// peersFileName := "peers.json"
		// configFileName := "config.json"

		// os.Rename(path.Join(nodeDataDir, identityFileName), path.Join(newDataContentDir, identityFileName))
		// os.Rename(path.Join(nodeDataDir, peersFileName), path.Join(newDataContentDir, peersFileName))
		// os.Rename(path.Join(nodeDataDir, configFileName), path.Join(newDataContentDir, configFileName))

		// err = os.RemoveAll(nodeDataDir)
		// util.AssertEE(err, "Failed to remove old node data directory!", constants.ExitIOError)
		// err = os.Rename(newDataContentDir, nodeDataDir)
		// util.AssertEE(err, "Failed to deploy new data directory!", constants.ExitIOError)

		// nodeDef, _, err := bb.Node.LoadAppDefinition()
		// util.AssertEE(err, "Failed to load node definition!", constants.ExitAppConfigurationLoadFailed)
		// nodeUser, ok := nodeDef["user"].(string)
		// util.AssertBE(ok, "Failed to get username from node!", constants.ExitInvalidUser)

		// log.Info("Upgrading storage...")
		// exitCode, err := bb_module_node.Module.UpgradeStorage()
		// util.AssertEE(err, "Failed to upgrade tezos storage", exitCode)

		// util.ChownR(nodeUser, path.Join(bb.Node.GetPath(), "data"))

		// os.RemoveAll(tmpDir)

		// if wasRunning {
		// 	bb.Node.Start()
		// }
		// log.Info("Node bootstrapped from tarball successfully...")
	},
}

func init() {
	bootstrapNodeCmd.Flags().Bool("no-check", false, "bootstraps node from tarball without veryfing snapshot integrity")
	RootCmd.AddCommand(bootstrapNodeCmd)
}
