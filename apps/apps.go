package apps

import (
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/apps/node"
	"github.com/tez-capital/tezbake/apps/peak"
	"github.com/tez-capital/tezbake/apps/signer"
)

var (
	Node   = node.FromPath("")
	Signer = signer.FromPath("")
	Peak   = peak.FromPath("")
	All    = []base.BakeBuddyApp{
		Node, Signer, Peak,
	}
	Implicit = []base.BakeBuddyApp{
		Node, Signer,
	}
)

type SetupContext = base.SetupContext
type UpgradeContext = base.UpgradeContext

type NodeInfoCollectionOptions = node.InfoCollectionOptions
type SignerInfoCollectionOptions = signer.InfoCollectionOptions

func GetInstalledApps() []base.BakeBuddyApp {
	result := make([]base.BakeBuddyApp, 0)
	for _, v := range All {
		if v.IsInstalled() {
			result = append(result, v)
		}
	}
	return result
}

func NodeFromPath(path string) *node.Node {
	return node.FromPath(path)
}

func SignerFromPath(path string) *signer.Signer {
	return signer.FromPath(path)
}

func PeakFromPath(path string) *peak.Peak {
	return peak.FromPath(path)
}
