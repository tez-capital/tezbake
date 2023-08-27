package bb

import (
	bb_module "alis.is/bb-cli/bb/modules"
	bb_module_node "alis.is/bb-cli/bb/modules/node"
	bb_module_signer "alis.is/bb-cli/bb/modules/signer"
)

const (
	VERSION = "0.9.1-beta"
)

var (
	Node    = bb_module_node.Module
	Signer  = bb_module_signer.Module
	Modules = []bb_module.IBakeBuddyModuleControl{
		Node, Signer,
	}
)

func GetInstalledModules() []bb_module.IBakeBuddyModuleControl {
	result := make([]bb_module.IBakeBuddyModuleControl, 0)
	for _, v := range Modules {
		if v.IsInstalled() {
			result = append(result, v)
		}
	}
	return result
}
