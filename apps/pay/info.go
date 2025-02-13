package pay

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
	"github.com/jedib0t/go-pretty/v6/text"
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
	return args
}

func (nico *InfoCollectionOptions) All() bool {
	return true
}

func (app *Tezpay) getInfoCollectionOptions(optionsJson []byte) *InfoCollectionOptions {
	result := &InfoCollectionOptions{}
	json.Unmarshal(optionsJson, result)
	return result
}

func (app *Tezpay) GetAvailableInfoCollectionOptions() []base.AmiInfoCollectionOption {
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

func (app *Tezpay) GetInfoFromOptions(options *InfoCollectionOptions) (map[string]interface{}, error) {
	args := options.toAmiArgs()
	infoBytes, _, err := ami.ExecuteInfo(app.GetPath(), args...)
	if err != nil {
		return base.GenerateFailedInfo(string(infoBytes), err), fmt.Errorf("failed to collect app info (%s)", err.Error())
	}

	return base.ParseInfoOutput[any](infoBytes)
}

func (app *Tezpay) GetInfo(optionsJson []byte) (map[string]interface{}, error) {
	return app.GetInfoFromOptions(app.getInfoCollectionOptions(optionsJson))
}

func (app *Tezpay) GetServiceInfo() (map[string]base.AmiServiceInfo, error) {
	result := map[string]base.AmiServiceInfo{}

	info, err := app.GetInfoFromOptions(&InfoCollectionOptions{Services: true})
	if err != nil {
		return result, err
	}

	jsonString, _ := json.Marshal(info["services"])
	json.Unmarshal(jsonString, &result)
	return result, err
}

func (app *Tezpay) IsServiceStatus(id string, status string) (bool, error) {
	serviceInfo, err := app.GetServiceInfo()
	if err != nil {
		return false, err
	}
	if service, ok := serviceInfo[constants.NodeAppServiceId]; ok && service.Status == status {
		return true, nil
	}
	return false, nil
}

func (app *Tezpay) PrintInfo(optionsJson []byte) error {
	tezpayInfo, err := app.GetInfo(optionsJson)
	if err != nil {
		return err
	}

	tezpayTable := table.NewWriter()
	tezpayTable.SetStyle(table.StyleLight)
	tezpayTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	tezpayTable.SetOutputMirror(os.Stdout)
	tezpayTable.AppendHeader(table.Row{app.GetLabel(), app.GetLabel()}, table.RowConfig{AutoMerge: true})

	tezpayTable.AppendRow(table.Row{"Status", fmt.Sprint(tezpayInfo["status"])})
	tezpayTable.AppendRow(table.Row{"Status Level", fmt.Sprint(tezpayInfo["level"])})

	tezpayTable.AppendSeparator()
	tezpayTable.AppendRow(table.Row{"Services", "Services"}, table.RowConfig{AutoMerge: true})
	tezpayTable.AppendSeparator()
	tezpayTable.AppendRow(table.Row{"Name", "Status (Started)"})
	tezpayTable.AppendSeparator()

	var services map[string]base.AmiServiceInfo
	jsonString, _ := json.Marshal(tezpayInfo["services"])
	json.Unmarshal(jsonString, &services)

	for k, v := range services {
		tezpayTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
	}

	tezpayTable.Render()
	return nil
}
