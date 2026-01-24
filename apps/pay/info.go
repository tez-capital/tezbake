package pay

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

type Info struct {
	base.InfoBase
	Services map[string]base.AmiServiceInfo `json:"services"`
	Type     string                         `json:"type"`
	Version  string                         `json:"version"`
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

type InfoCollectionOptions struct {
	Timeout  int
	Services bool
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

func (app *Tezpay) GetInfoFromOptions(options *InfoCollectionOptions) (Info, error) {
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

func (app *Tezpay) GetInfo(optionsJson []byte) (any, error) {
	return app.GetInfoFromOptions(app.getInfoCollectionOptions(optionsJson))
}

func (app *Tezpay) GetServiceInfo() (map[string]base.AmiServiceInfo, error) {
	result := map[string]base.AmiServiceInfo{}

	info, err := app.GetInfoFromOptions(&InfoCollectionOptions{Services: true})
	if err != nil {
		return result, err
	}

	return info.Services, err
}

func (app *Tezpay) IsServiceStatus(id string, status string) (bool, error) {
	return base.IsServiceStatus(app, id, status)
}

func (app *Tezpay) IsAnyServiceStatus(status string) (bool, error) {
	return base.IsAnyServiceStatus(app, status)
}

func (app *Tezpay) PrintInfo(optionsJson []byte) error {
	tezpayInfoRaw, err := app.GetInfo(optionsJson)
	if err != nil {
		return err
	}
	tezpayInfo, ok := tezpayInfoRaw.(Info)
	if !ok {
		return fmt.Errorf("invalid tezpay info type")
	}

	tezpayTable := table.NewWriter()
	tezpayTable.SetStyle(table.StyleLight)
	tezpayTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	tezpayTable.SetOutputMirror(os.Stdout)
	tezpayTable.AppendHeader(table.Row{app.GetLabel(), app.GetLabel()}, table.RowConfig{AutoMerge: true})

	tezpayTable.AppendRow(table.Row{"Status", tezpayInfo.Status})
	tezpayTable.AppendRow(table.Row{"Status Level", tezpayInfo.Level})

	tezpayTable.AppendSeparator()
	tezpayTable.AppendRow(table.Row{"Services", "Services"}, table.RowConfig{AutoMerge: true})
	tezpayTable.AppendSeparator()
	tezpayTable.AppendRow(table.Row{"Name", "Status (Started)"})
	tezpayTable.AppendSeparator()

	for k, v := range tezpayInfo.Services {
		tezpayTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
	}

	tezpayTable.Render()
	return nil
}
