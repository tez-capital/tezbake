package apps

import (
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/apps/dal"
	"github.com/tez-capital/tezbake/apps/node"
	"github.com/tez-capital/tezbake/apps/pay"
	"github.com/tez-capital/tezbake/apps/peak"
	"github.com/tez-capital/tezbake/apps/signer"
)

var (
	Node    = node.FromPath("")
	DalNode = dal.FromPath("")
	Signer  = signer.FromPath("")
	Peak    = peak.FromPath("")
	Pay     = pay.FromPath("")
	All     = []base.BakeBuddyApp{
		Node, Signer, DalNode, Peak, Pay,
	}
	Implicit = []base.BakeBuddyApp{
		Node, Signer,
	}
)

type SetupContext = base.SetupContext
type UpgradeContext = base.UpgradeContext

type NodeInfoCollectionOptions = node.InfoCollectionOptions
type DalNodeInfoCollectionOptions = dal.InfoCollectionOptions
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

func DalNodeFromPath(path string) *dal.DalNode {
	return dal.FromPath(path)
}

func SignerFromPath(path string) *signer.Signer {
	return signer.FromPath(path)
}

func PeakFromPath(path string) *peak.Peak {
	return peak.FromPath(path)
}

func TezpayFromPath(path string) *pay.Tezpay {
	return pay.FromPath(path)
}
