package signer

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/samber/lo"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps/base"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type InfoCollectionOptions struct {
	//Timeout  int
	Wallets  bool
	Simple   bool
	Services bool
	// Sensitive bool // Not needed for now
}

func (infoCollectionOptions *InfoCollectionOptions) toAmiArgs() []string {
	args := make([]string, 0)
	// if infoCollectionOptions.Timeout > 0 {
	// 	args = append(args, fmt.Sprintf("--timeout=%d", infoCollectionOptions.Timeout))
	// }

	if infoCollectionOptions.Wallets {
		args = append(args, "--wallets")
	}
	if infoCollectionOptions.Simple {
		args = append(args, "--simple")
	}
	if infoCollectionOptions.Services {
		args = append(args, "--services")
	}
	// if infoCollectionOptions.Sensitive {
	// 	args = append(args, "--sensitive")
	// }
	return args
}

func (sico *InfoCollectionOptions) All() bool {
	return !sico.Wallets && !sico.Simple && !sico.Services
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
	return base.ParseInfoOutput[any](infoBytes)
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

	if infoCollectionOptions.All() || infoCollectionOptions.Simple || infoCollectionOptions.Wallets {
		// Baker Info
		signerTable.AppendSeparator()
		signerTable.AppendRow(table.Row{"Wallets", "Wallets"}, table.RowConfig{AutoMerge: true})
		signerTable.AppendSeparator()
		if wallets, ok := signerInfo["wallets"].(map[string]interface{}); ok {
			wallet_ids := lo.Keys(wallets)
			sort.Strings(wallet_ids)
			for _, k := range wallet_ids {
				v := wallets[k]
				if properties, ok := v.(map[string]interface{}); ok {
					kind := fmt.Sprint(properties["kind"])
					switch kind {
					case "ledger":
						status := "error"
						if properties["ledger_status"] == "connected" && properties["authorized"] == true {
							status = "ok"
						}
						signerTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v) - %v", kind, properties["pkh"], status)})
					case "soft":
						signerTable.AppendRow(table.Row{k, fmt.Sprintf("⚠️ %v ⚠️ (%v)", kind, properties["pkh"])})
					case "remote":
						signerTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", kind, properties["pkh"])})
					}
				} else {
					signerTable.AppendRow(table.Row{k, "unknown data format"})
				}
			}
		} else {
			signerTable.AppendRow(table.Row{"N/A", "N/A"})
		}
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
