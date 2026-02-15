package node

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"

	"github.com/jedib0t/go-pretty/v6/table"
)

type Info struct {
	base.InfoBase
	AdditionalBakingKeys map[string]string              `json:"additional_baking_keys"`
	Bootstrapped         bool                           `json:"bootstrapped"`
	ChainHead            ChainHeadInfo                  `json:"chain_head"`
	Connections          int                            `json:"connections"`
	Services             map[string]base.AmiServiceInfo `json:"services"`
	SyncState            string                         `json:"sync_state"`
	Type                 string                         `json:"type"`
	Version              string                         `json:"version"`
	VotingCurrentPeriod  VotingCurrentPeriod            `json:"voting_current_period"`
	VotingProposals      []any                          `json:"voting_proposals"`
	IsRemote             bool                           `json:"isRemote"`
}

func (i *Info) UnmarshalJSON(data []byte) error {
	type Alias Info
	aux := &struct {
		Services json.RawMessage `json:"services"`
		*Alias
	}{
		Alias: (*Alias)(i),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := base.UnmarshalIfNotEmptyArray(aux.Services, &i.Services); err != nil {
		return err
	}

	return nil
}

type ChainHeadInfo struct {
	Cycle        int    `json:"cycle"`
	Hash         string `json:"hash"`
	Level        int    `json:"level"`
	Protocol     string `json:"protocol"`
	ProtocolNext string `json:"protocol_next"`
	Timestamp    string `json:"timestamp"`
}

type VotingCurrentPeriod struct {
	Position     int          `json:"position"`
	Remaining    int          `json:"remaining"`
	VotingPeriod VotingPeriod `json:"voting_period"`
}

type VotingPeriod struct {
	Index         int    `json:"index"`
	Kind          string `json:"kind"`
	StartPosition int    `json:"start_position"`
}

type InfoCollectionOptions struct {
	Timeout  int
	Chain    bool
	Simple   bool
	Services bool
	Voting   bool
}

func (infoCollectionOptions *InfoCollectionOptions) toAmiArgs() []string {
	args := make([]string, 0)
	if infoCollectionOptions.Timeout > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", infoCollectionOptions.Timeout))
	}

	if infoCollectionOptions.Chain {
		args = append(args, "--chain")
	}
	if infoCollectionOptions.Simple {
		args = append(args, "--simple")
	}
	if infoCollectionOptions.Services {
		args = append(args, "--services")
	}
	if infoCollectionOptions.Voting {
		args = append(args, "--voting")
	}
	return args
}

func (nico *InfoCollectionOptions) All() bool {
	return !nico.Chain && !nico.Simple && !nico.Services && !nico.Voting
}

func (app *Node) getInfoCollectionOptions(optionsJson []byte) *InfoCollectionOptions {
	result := &InfoCollectionOptions{}
	json.Unmarshal(optionsJson, result)
	return result
}

func (app *Node) GetAvailableInfoCollectionOptions() []base.AmiInfoCollectionOption {
	result := make([]base.AmiInfoCollectionOption, 0)
	options := InfoCollectionOptions{}
	val := reflect.ValueOf(options)

	for i := 0; i < val.NumField(); i++ {
		result = append(result, base.AmiInfoCollectionOption{
			Name: strings.ToLower(val.Type().Field(i).Name),
			Type: strings.ToLower(val.Type().Field(i).Type.Name()),
		})
	}
	return result
}

func (app *Node) GetInfoFromOptions(options *InfoCollectionOptions) (Info, error) {
	args := options.toAmiArgs()
	infoBytes, _, err := ami.ExecuteInfo(app.GetPath(), args...)
	if err != nil {
		failedInfo := Info{
			InfoBase: base.GenerateFailedInfo(string(infoBytes), err),
		}
		return failedInfo, fmt.Errorf("failed to collect app info (%s)", err.Error())
	}

	info, err := base.ParseInfoOutput[Info](infoBytes)
	info.IsRemote, _ = ami.IsRemoteApp(app.GetPath())
	if err != nil {
		return Info{InfoBase: base.GenerateFailedInfo(string(infoBytes), err)}, err
	}
	return info, err
}

func (app *Node) GetInfo(optionsJson []byte) (any, error) {
	return app.GetInfoFromOptions(app.getInfoCollectionOptions(optionsJson))
}

func (app *Node) GetServiceInfo() (map[string]base.AmiServiceInfo, error) {
	result := map[string]base.AmiServiceInfo{}

	info, err := app.GetInfoFromOptions(&InfoCollectionOptions{Services: true})
	if err != nil {
		return result, err
	}

	return info.Services, err
}

func (app *Node) IsServiceStatus(id string, status string) (bool, error) {
	return base.IsServiceStatus(app, id, status)
}

func (app *Node) IsAnyServiceStatus(status string) (bool, error) {
	return base.IsAnyServiceStatus(app, status)
}

func (app *Node) PrintInfo(optionsJson []byte) error {
	nodeInfoRaw, err := app.GetInfo(optionsJson)
	if err != nil {
		return err
	}
	nodeInfo, ok := nodeInfoRaw.(Info)
	if !ok {
		return fmt.Errorf("invalid signer info type")
	}

	infoCollectionOptions := app.getInfoCollectionOptions(optionsJson)

	nodeTable := table.NewWriter()
	nodeTable.SetStyle(table.StyleLight)
	nodeTable.SetOutputMirror(os.Stdout)

	nodeTable.AppendHeader(table.Row{app.GetLabel(), app.GetLabel()}, table.RowConfig{AutoMerge: true})

	if infoCollectionOptions.All() || infoCollectionOptions.Simple || (infoCollectionOptions.Services && infoCollectionOptions.Chain) {
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Status", nodeInfo.Status})
		nodeTable.AppendRow(table.Row{"Status Level", nodeInfo.Level})
		nodeTable.AppendRow(table.Row{"Bootstrapped", nodeInfo.Bootstrapped})
		nodeTable.AppendRow(table.Row{"Sync State", nodeInfo.SyncState})
		nodeTable.AppendRow(table.Row{"Connections", nodeInfo.Connections})
	}

	if infoCollectionOptions.All() || infoCollectionOptions.Simple || (infoCollectionOptions.Services && infoCollectionOptions.Chain) {
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Baking Keys", "Baking Keys"}, table.RowConfig{AutoMerge: true})
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Name", "Address"})
		nodeTable.AppendSeparator()

		if len(nodeInfo.AdditionalBakingKeys) == 0 {
			nodeTable.AppendRow(table.Row{"N/A", "N/A"})
		} else {
			bakingKeyNames := make([]string, 0, len(nodeInfo.AdditionalBakingKeys))
			for name := range nodeInfo.AdditionalBakingKeys {
				bakingKeyNames = append(bakingKeyNames, name)
			}
			sort.Strings(bakingKeyNames)
			for _, name := range bakingKeyNames {
				nodeTable.AppendRow(table.Row{name, nodeInfo.AdditionalBakingKeys[name]})
			}
		}
	}

	if chainInfo := nodeInfo.ChainHead; infoCollectionOptions.All() || infoCollectionOptions.Chain {
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Chain State", "Chain State"}, table.RowConfig{AutoMerge: true})
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Cycle", chainInfo.Cycle})
		nodeTable.AppendRow(table.Row{"Level", chainInfo.Level})
		nodeTable.AppendRow(table.Row{"Protocol", chainInfo.Protocol})
		nodeTable.AppendRow(table.Row{"Hash", chainInfo.Hash})
	}

	if infoCollectionOptions.All() || infoCollectionOptions.Simple || infoCollectionOptions.Services {
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Services", "Services"}, table.RowConfig{AutoMerge: true})
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Name", "Status (Started)"})
		nodeTable.AppendSeparator()

		for k, v := range nodeInfo.Services {
			nodeTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
		}
	}

	nodeTable.Render()
	return nil
}
