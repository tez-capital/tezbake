package node

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/constants"

	"github.com/jedib0t/go-pretty/v6/table"
)

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

func (app *Node) GetInfoFromOptions(options *InfoCollectionOptions) (map[string]interface{}, error) {
	args := options.toAmiArgs()
	infoBytes, _, err := ami.ExecuteInfo(app.GetPath(), args...)
	if err != nil {
		return base.GenerateFailedInfo(string(infoBytes), err), fmt.Errorf("failed to collect app info (%s)", err.Error())
	}

	info, err := base.ParseInfoOutput(infoBytes)
	info["isRemote"], _ = ami.IsRemoteApp(app.GetPath())
	return info, err
}

func (app *Node) GetInfo(optionsJson []byte) (map[string]interface{}, error) {
	return app.GetInfoFromOptions(app.getInfoCollectionOptions(optionsJson))
}

func (app *Node) GetServiceInfo() (map[string]base.AmiServiceInfo, error) {
	result := map[string]base.AmiServiceInfo{}

	info, err := app.GetInfoFromOptions(&InfoCollectionOptions{Services: true})
	if err != nil {
		return result, err
	}

	jsonString, _ := json.Marshal(info["services"])
	json.Unmarshal(jsonString, &result)
	return result, err
}

func (app *Node) IsServiceStatus(id string, status string) (bool, error) {
	serviceInfo, err := app.GetServiceInfo()
	if err != nil {
		return false, err
	}
	if service, ok := serviceInfo[constants.NodeAppServiceId]; ok && service.Status == status {
		return true, nil
	}
	return false, nil
}

func (app *Node) PrintInfo(optionsJson []byte) error {
	nodeInfo, err := app.GetInfo(optionsJson)
	if err != nil {
		return err
	}
	infoCollectionOptions := app.getInfoCollectionOptions(optionsJson)

	nodeTable := table.NewWriter()
	nodeTable.SetStyle(table.StyleLight)
	nodeTable.SetOutputMirror(os.Stdout)

	nodeTable.AppendHeader(table.Row{app.GetLabel(), app.GetLabel()}, table.RowConfig{AutoMerge: true})

	if infoCollectionOptions.All() || infoCollectionOptions.Simple || (infoCollectionOptions.Services && infoCollectionOptions.Chain) {
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Status", fmt.Sprint(nodeInfo["status"])})
		nodeTable.AppendRow(table.Row{"Status Level", fmt.Sprint(nodeInfo["level"])})
		nodeTable.AppendRow(table.Row{"Bootstrapped", fmt.Sprint(nodeInfo["bootstrapped"])})
		nodeTable.AppendRow(table.Row{"Sync State", fmt.Sprint(nodeInfo["sync_state"])})
		nodeTable.AppendRow(table.Row{"Connections", fmt.Sprint(nodeInfo["connections"])})
	}

	if chainInfo, ok := nodeInfo["chain_head"].(map[string]interface{}); ok && (infoCollectionOptions.All() || infoCollectionOptions.Chain) {
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Chain State", "Chain State"}, table.RowConfig{AutoMerge: true})
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Cycle", fmt.Sprint(chainInfo["cycle"])})
		nodeTable.AppendRow(table.Row{"Level", fmt.Sprint(int(chainInfo["level"].(float64)))})
		nodeTable.AppendRow(table.Row{"Protocol", fmt.Sprint(chainInfo["protocol"])})
		nodeTable.AppendRow(table.Row{"Hash", fmt.Sprint(chainInfo["hash"])})
	}

	if infoCollectionOptions.All() || infoCollectionOptions.Simple || infoCollectionOptions.Services {
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Services", "Services"}, table.RowConfig{AutoMerge: true})
		nodeTable.AppendSeparator()
		nodeTable.AppendRow(table.Row{"Name", "Status (Started)"})
		nodeTable.AppendSeparator()

		var services map[string]base.AmiServiceInfo
		jsonString, _ := json.Marshal(nodeInfo["services"])
		json.Unmarshal(jsonString, &services)

		for k, v := range services {
			nodeTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
		}
	}

	nodeTable.Render()
	return nil
}
