package signer

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type InfoCollectionOptions struct {
	//Timeout  int
	Baking   bool
	Simple   bool
	Services bool
}

func (infoCollectionOptions *InfoCollectionOptions) toAmiArgs() []string {
	args := make([]string, 0)
	// if infoCollectionOptions.Timeout > 0 {
	// 	args = append(args, fmt.Sprintf("--timeout=%d", infoCollectionOptions.Timeout))
	// }

	if infoCollectionOptions.Baking {
		args = append(args, "--baking")
	}
	if infoCollectionOptions.Simple {
		args = append(args, "--simple")
	}
	if infoCollectionOptions.Services {
		args = append(args, "--services")
	}
	return args
}

func (sico *InfoCollectionOptions) All() bool {
	return !sico.Baking && !sico.Simple && !sico.Services
}

func (app *Signer) getInfoCollectionOptions(optionsJson []byte) *InfoCollectionOptions {
	result := &InfoCollectionOptions{}
	json.Unmarshal(optionsJson, result)
	return result
}

func (app *Signer) GetAvailableInfoCollectionOptions() []base.AmiInfoCollectionOption {
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

func (app *Signer) GetInfoFromOptions(options *InfoCollectionOptions) (map[string]interface{}, error) {
	args := options.toAmiArgs()
	infoBytes, _, err := ami.ExecuteInfo(app.GetPath(), args...)
	if err != nil {
		return base.GenerateFailedInfo(string(infoBytes), err), fmt.Errorf("failed to collect app info (%s)", err.Error())
	}
	return base.ParseInfoOutput(infoBytes)
}

func (app *Signer) GetInfo(optionsJson []byte) (map[string]interface{}, error) {
	return app.GetInfoFromOptions(app.getInfoCollectionOptions(optionsJson))
}

func (app *Signer) GetServiceInfo() (map[string]base.AmiServiceInfo, error) {
	result := map[string]base.AmiServiceInfo{}

	info, err := app.GetInfoFromOptions(&InfoCollectionOptions{Services: true})
	if err != nil {
		return result, err
	}
	jsonString, _ := json.Marshal(info["services"])
	json.Unmarshal(jsonString, &result)
	return result, err
}

func (app *Signer) IsServiceStatus(id string, status string) (bool, error) {
	serviceInfo, err := app.GetServiceInfo()
	if err != nil {
		return false, err
	}
	if service, ok := serviceInfo[id]; ok && service.Status == status {
		return true, nil
	}
	return false, nil
}

func (app *Signer) PrintInfo(optionsJson []byte) error {
	signerInfo, err := app.GetInfo(optionsJson)
	if err != nil {
		return err
	}

	infoCollectionOptions := app.getInfoCollectionOptions(optionsJson)

	signerTable := table.NewWriter()
	signerTable.SetStyle(table.StyleLight)
	signerTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	signerTable.SetOutputMirror(os.Stdout)
	signerTable.AppendHeader(table.Row{app.GetLabel(), app.GetLabel()}, table.RowConfig{AutoMerge: true})

	signerTable.AppendRow(table.Row{"Status", fmt.Sprint(signerInfo["status"])})
	signerTable.AppendRow(table.Row{"Status Level", fmt.Sprint(signerInfo["level"])})

	if infoCollectionOptions.All() || infoCollectionOptions.Simple || infoCollectionOptions.Baking {
		// Baker Info
		signerTable.AppendSeparator()
		signerTable.AppendRow(table.Row{"Baking", "Baking"}, table.RowConfig{AutoMerge: true})
		signerTable.AppendSeparator()
		signerTable.AppendRow(table.Row{"Ledger Id", fmt.Sprint(signerInfo["ledger_id"])})
		signerTable.AppendRow(table.Row{"Baking App", fmt.Sprint(signerInfo["baking_app"])})
		signerTable.AppendRow(table.Row{"Baking App Status", fmt.Sprint(signerInfo["baking_app_status"])})
		signerTable.AppendRow(table.Row{"Baker Address", fmt.Sprint(signerInfo["baker_address"])})
	}

	if infoCollectionOptions.All() || infoCollectionOptions.Services {
		signerTable.AppendSeparator()
		signerTable.AppendRow(table.Row{"Services", "Services"}, table.RowConfig{AutoMerge: true})
		signerTable.AppendSeparator()
		signerTable.AppendRow(table.Row{"Name", "Status (Started)"})
		signerTable.AppendSeparator()

		var services map[string]base.AmiServiceInfo
		jsonString, _ := json.Marshal(signerInfo["services"])
		json.Unmarshal(jsonString, &services)

		for k, v := range services {
			signerTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
		}
	}
	signerTable.Render()
	return nil
}
