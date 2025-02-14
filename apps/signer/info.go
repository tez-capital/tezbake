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

type Info struct {
	base.InfoBase
	Services map[string]base.AmiServiceInfo `json:"services"`
	Type     string                         `json:"type"`
	Version  string                         `json:"version"`
	Wallets  map[string]base.AmiWalletInfo  `json:"wallets"`
}

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

func (app *Signer) GetInfoFromOptions(options *InfoCollectionOptions) (Info, error) {
	args := options.toAmiArgs()
	infoBytes, _, err := ami.ExecuteInfo(app.GetPath(), args...)
	if err != nil {
		failedInfo := Info{
			InfoBase: base.GenerateFailedInfo(string(infoBytes), err),
		}
		return failedInfo, fmt.Errorf("failed to collect app info (%s)", err.Error())
	}

	info, err := base.ParseInfoOutput[Info](infoBytes)
	if err != nil {
		return Info{InfoBase: base.GenerateFailedInfo(string(infoBytes), err)}, err
	}
	return info, nil
}

func (app *Signer) GetInfo(optionsJson []byte) (any, error) {
	return app.GetInfoFromOptions(app.getInfoCollectionOptions(optionsJson))
}

func (app *Signer) GetServiceInfo() (map[string]base.AmiServiceInfo, error) {
	result := map[string]base.AmiServiceInfo{}

	info, err := app.GetInfoFromOptions(&InfoCollectionOptions{Services: true})
	if err != nil {
		return result, err
	}
	return info.Services, err
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
	signerInfoRaw, err := app.GetInfo(optionsJson)
	if err != nil {
		return err
	}
	signerInfo, ok := signerInfoRaw.(Info)
	if !ok {
		return fmt.Errorf("invalid signer info type")
	}

	infoCollectionOptions := app.getInfoCollectionOptions(optionsJson)

	signerTable := table.NewWriter()
	signerTable.SetStyle(table.StyleLight)
	signerTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	signerTable.SetOutputMirror(os.Stdout)
	signerTable.AppendHeader(table.Row{app.GetLabel(), app.GetLabel()}, table.RowConfig{AutoMerge: true})

	signerTable.AppendRow(table.Row{"Status", signerInfo.Status})
	signerTable.AppendRow(table.Row{"Status Level", signerInfo.Level})

	if infoCollectionOptions.All() || infoCollectionOptions.Simple || infoCollectionOptions.Wallets {
		// Baker Info
		signerTable.AppendSeparator()
		signerTable.AppendRow(table.Row{"Wallets", "Wallets"}, table.RowConfig{AutoMerge: true})
		signerTable.AppendSeparator()
		if wallets := signerInfo.Wallets; len(wallets) > 0 {
			wallet_ids := lo.Keys(wallets)
			sort.Strings(wallet_ids)
			for _, k := range wallet_ids {
				walletProperties := wallets[k]

				kind := walletProperties.Kind
				pkh := walletProperties.Pkh
				switch kind {
				case "ledger":
					status := "error"
					if walletProperties.LedgerStatus == "connected" && walletProperties.Authorized == true {
						status = "ok"
					}
					signerTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v) - %v", kind, pkh, status)})
				case "soft":
					signerTable.AppendRow(table.Row{k, fmt.Sprintf("⚠️ %v ⚠️ (%v)", kind, pkh)})
				case "remote":
					signerTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", kind, pkh)})
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

		for k, v := range signerInfo.Services {
			signerTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
		}
	}
	signerTable.Render()
	return nil
}
