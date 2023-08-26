package bb_module_signer

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"alis.is/bb-cli/ami"
	bb_module "alis.is/bb-cli/bb/modules"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type signerInfoCollectionOptions struct {
	//Timeout  int
	Baking   bool
	Simple   bool
	Services bool
}

func (infoCollectionOptions *signerInfoCollectionOptions) toAmiArgs() []string {
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

func (sico *signerInfoCollectionOptions) All() bool {
	return !sico.Baking && !sico.Simple && !sico.Services
}

func (app *Signer) getInfoCollectionOptions(optionsJson []byte) *signerInfoCollectionOptions {
	result := &signerInfoCollectionOptions{}
	json.Unmarshal(optionsJson, result)
	return result
}

func (app *Signer) GetAvailableInfoCollectionOptions() []bb_module.AmiInfoCollectionOption {
	result := make([]bb_module.AmiInfoCollectionOption, 0)
	options := signerInfoCollectionOptions{}
	val := reflect.ValueOf(options)
	for i := 0; i < val.NumField(); i++ {
		result = append(result, bb_module.AmiInfoCollectionOption{
			Name: strings.ToLower(val.Type().Field(i).Name),
			Type: strings.ToLower(val.Type().Field(i).Type.Name()),
		})
	}
	return result
}

func (app *Signer) GetInfo(optionsJson []byte) (map[string]interface{}, error) {
	args := app.getInfoCollectionOptions(optionsJson).toAmiArgs()
	infoBytes, _, err := ami.ExecuteInfo(app.GetPath(), args...)
	if err != nil {
		return bb_module.GenerateFailedInfo(string(infoBytes), err), fmt.Errorf("failed to collect app info (%s)", err.Error())
	}
	return bb_module.ParseInfoOutput(infoBytes)
}

func (app *Signer) GetServiceInfo() (map[string]bb_module.AmiServiceInfo, error) {
	result := map[string]bb_module.AmiServiceInfo{}

	args := (&signerInfoCollectionOptions{Services: true}).toAmiArgs()
	infoBytes, _, err := ami.ExecuteInfo(app.GetPath(), args...)
	if err != nil {
		return result, err
	}
	info, err := bb_module.ParseInfoOutput(infoBytes)
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

		if services, ok := signerInfo["services"].(map[string]bb_module.AmiServiceInfo); ok {
			for k, v := range services {
				signerTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
			}
		} else {
			// TODO: remove with new packages
			for k, v := range signerInfo {
				if strings.HasSuffix(k, "_started") {
					serviceId := k[:len(k)-len("_started")]
					if status, ok := signerInfo[serviceId]; ok {
						signerTable.AppendRow(table.Row{serviceId, fmt.Sprintf("%v (%v)", status, v)})
					}
				}
			}
		}
	}
	signerTable.Render()
	return nil
}
