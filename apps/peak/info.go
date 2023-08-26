package peak

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
	Services bool
}

func (infoCollectionOptions *InfoCollectionOptions) toAmiArgs() []string {
	args := make([]string, 0)
	if infoCollectionOptions.Services {
		args = append(args, "--services")
	}
	return args
}

func (sico *InfoCollectionOptions) All() bool {
	return true
}

func (app *Peak) getInfoCollectionOptions(optionsJson []byte) *InfoCollectionOptions {
	result := &InfoCollectionOptions{}
	json.Unmarshal(optionsJson, result)
	return result
}

func (app *Peak) GetAvailableInfoCollectionOptions() []base.AmiInfoCollectionOption {
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

func (app *Peak) GetInfoFromOptions(options *InfoCollectionOptions) (map[string]interface{}, error) {
	args := options.toAmiArgs()
	infoBytes, _, err := ami.ExecuteInfo(app.GetPath(), args...)
	if err != nil {
		return base.GenerateFailedInfo(string(infoBytes), err), fmt.Errorf("failed to collect app info (%s)", err.Error())
	}
	return base.ParseInfoOutput(infoBytes)
}

func (app *Peak) GetInfo(optionsJson []byte) (map[string]interface{}, error) {
	return app.GetInfoFromOptions(app.getInfoCollectionOptions(optionsJson))
}

func (app *Peak) GetServiceInfo() (map[string]base.AmiServiceInfo, error) {
	result := map[string]base.AmiServiceInfo{}

	info, err := app.GetInfoFromOptions(&InfoCollectionOptions{Services: true})
	if err != nil {
		return result, err
	}
	jsonString, _ := json.Marshal(info["services"])
	json.Unmarshal(jsonString, &result)
	return result, err
}

func (app *Peak) IsServiceStatus(id string, status string) (bool, error) {
	serviceInfo, err := app.GetServiceInfo()
	if err != nil {
		return false, err
	}
	if service, ok := serviceInfo[id]; ok && service.Status == status {
		return true, nil
	}
	return false, nil
}

func (app *Peak) PrintInfo(optionsJson []byte) error {
	peakInfo, err := app.GetInfo(optionsJson)
	if err != nil {
		return err
	}

	peakTable := table.NewWriter()
	peakTable.SetStyle(table.StyleLight)
	peakTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	peakTable.SetOutputMirror(os.Stdout)
	peakTable.AppendHeader(table.Row{app.GetLabel(), app.GetLabel()}, table.RowConfig{AutoMerge: true})

	peakTable.AppendRow(table.Row{"Status", fmt.Sprint(peakInfo["status"])})
	peakTable.AppendRow(table.Row{"Status Level", fmt.Sprint(peakInfo["level"])})

	peakTable.AppendSeparator()
	peakTable.AppendRow(table.Row{"Services", "Services"}, table.RowConfig{AutoMerge: true})
	peakTable.AppendSeparator()
	peakTable.AppendRow(table.Row{"Name", "Status (Started)"})
	peakTable.AppendSeparator()

	var services map[string]base.AmiServiceInfo
	jsonString, _ := json.Marshal(peakInfo["services"])
	json.Unmarshal(jsonString, &services)

	for k, v := range services {
		peakTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
	}

	peakTable.Render()
	return nil
}
