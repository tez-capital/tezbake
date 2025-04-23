package base

import "github.com/tez-capital/tezbake/ami"

const (
	MergingSetupKind     = "merge"
	OverWritingSetupKind = "overwrite"
)

type BakeBuddyApp interface {
	GetId() string
	GetLabel() string
	GetPath() string
	IsRemoteApp() bool
	Execute(args ...string) (int, error)
	GetAmiTemplate(ctx *SetupContext) map[string]interface{}
	Setup(ctx *SetupContext, args ...string) (int, error)
	Upgrade(ctx *UpgradeContext, args ...string) (int, error)
	Stop(args ...string) (int, error)
	Start(args ...string) (int, error)
	Remove(all bool, args ...string) (int, error)
	LoadAppDefinition() (map[string]interface{}, string, error)
	LoadAppConfiguration() (map[string]interface{}, error)
	GetAvailableInfoCollectionOptions() []AmiInfoCollectionOption
	GetInfo(optionsJson []byte) (any, error)
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

type AmiWalletInfo struct {
	Authorized    bool   `json:"authorized,omitempty"`
	AppVersion    string `json:"app_version,omitempty"`
	DeviceAddress string `json:"device_address,omitempty"`
	DeviceBus     string `json:"device_bus,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Ledger        string `json:"ledger,omitempty"`
	LedgerStatus  string `json:"ledger_status,omitempty"`
	Pkh           string `json:"pkh,omitempty"`
}

type BBInstanceVersions struct {
	Cli           string                `json:"cli"`
	RemoteCli     string                `json:"remote-cli"`
	Node          *ami.InstanceVersions `json:"node"`
	Signer        *ami.InstanceVersions `json:"signer"`
	Dal           *ami.InstanceVersions `json:"dal"`
	Peak          *ami.InstanceVersions `json:"peak"`
	Pay           *ami.InstanceVersions `json:"pay"`
	HasRemoteNode bool                  `json:"has-remote-node"`
}

type UpgradeContext struct {
	UpgradeStorage bool `json:"upgrade-storage,omitempty"`
}
