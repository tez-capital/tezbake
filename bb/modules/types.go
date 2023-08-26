package bb_module

import "alis.is/bb-cli/ami"

const (
	MergingSetupKind     = "merge"
	OverWritingSetupKind = "overwrite"
)

type IBakeBuddyModuleControl interface {
	GetId() string
	GetLabel() string
	GetPath() string
	GetAmiTemplate(ctx *SetupContext) map[string]interface{}
	Setup(ctx *SetupContext, args ...string) (int, error)
	Upgrade(ctx *UpgradeContext, args ...string) (int, error)
	Stop(args ...string) (int, error)
	Start(args ...string) (int, error)
	Remove(all bool, args ...string) (int, error)
	LoadAppDefinition() (map[string]interface{}, string, error)
	LoadAppConfiguration() (map[string]interface{}, error)
	GetAvailableInfoCollectionOptions() []AmiInfoCollectionOption
	GetInfo(optionsJson []byte) (map[string]interface{}, error)
	GetServiceInfo() (map[string]AmiServiceInfo, error)
	PrintInfo(optionsJson []byte) error
	GetVersions(options *ami.CollectVersionsOptions) (*ami.InstanceVersions, error)
	GetAppVersion() (string, error)
	IsInstalled() bool
	SupportsRemote() bool
	GetSetupKind() string
}

type AmiInfoCollectionOption struct {
	Name string
	Type string
}

type AmiServiceInfo struct {
	Status  string `json:"status"`
	Started string `json:"started"`
}

type BBInstanceVersions struct {
	Cli           string                `json:"cli"`
	RemoteCli     string                `json:"remote-cli"`
	Node          *ami.InstanceVersions `json:"node"`
	Signer        *ami.InstanceVersions `json:"signer"`
	HasRemoteNode bool                  `json:"has-remote-node"`
}
