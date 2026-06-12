package base

import (
	"os"

	"github.com/tez-capital/tezbake/ami"
)

const (
	MergingSetupKind     = "merge"
	OverWritingSetupKind = "overwrite"
)

type BakeBuddyApp interface {
	GetId() string
	GetLabel() string
	GetPath() string
	GetLocalPath() string
	IsRemoteApp() bool
	Execute(args ...string) (int, error)
	GetAmiTemplate(ctx *SetupContext) map[string]any
	Setup(ctx *SetupContext, args ...string) (int, error)
	Upgrade(ctx *UpgradeContext, args ...string) (int, error)
	Stop(args ...string) (int, error)
	Start(args ...string) (int, error)
	Remove(all bool, args ...string) (int, error)
	LoadAppDefinition() (map[string]any, string, error)
	LoadAppConfiguration() (map[string]any, error)
	GetAvailableInfoCollectionOptions() []AmiInfoCollectionOption
	GetInfo(optionsJson []byte) (any, error)
	GetServiceInfo() (map[string]AmiServiceInfo, error)
	PrintInfo(optionsJson []byte) error
	GetVersions(options ami.CollectVersionsOptions) (*ami.InstanceVersions, error)
	GetVersion() (string, error)
	IsInstalled() bool
	Pack(ctx *PackContext, output string) (int, error)
	Unpack(ctx *UnpackContext, source string) (int, error)
}

type PackContext struct {
	Mode string `json:"mode,omitempty"`
}

type RemoteBundleFile struct {
	Data []byte
	Mode os.FileMode
}

type RemoteBundle struct {
	Locator *ami.RemoteConfiguration
	Files   map[string]RemoteBundleFile
}

type RemoteUnpackOverride struct {
	Remote        string
	RemoteAuth    string
	RemoteElevate ami.RemoteElevationKind
	RemotePath    string
}

func (o RemoteUnpackOverride) Active() bool {
	return o.Remote != "" || o.RemoteAuth != "" || o.RemoteElevate != "" || o.RemotePath != ""
}

type UnpackContext struct {
	RemoteBundle   *RemoteBundle
	RemoteOverride RemoteUnpackOverride
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
	Authorized   bool   `json:"authorized,omitempty"`
	AppVersion   string `json:"app_version,omitempty"`
	DevicePath   string `json:"device_path,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Ledger       string `json:"ledger,omitempty"`
	LedgerStatus string `json:"ledger_status,omitempty"`
	Pkh          string `json:"pkh,omitempty"`
	Status       string `json:"status,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"`
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
	UpgradeStorage bool   `json:"upgrade-storage,omitempty"`
	Branch         string
	Version        string
}
