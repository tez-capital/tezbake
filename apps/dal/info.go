package dal

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
	Services         map[string]base.AmiServiceInfo `json:"services"`
	Type             string                         `json:"type"`
	Version          string                         `json:"version"`
	AttesterProfiles []string                       `json:"attester_profiles"`
	IsRemote         bool                           `json:"isRemote"`
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
	Dal      bool
}

func (infoCollectionOptions *InfoCollectionOptions) toAmiArgs() []string {
	args := make([]string, 0)
	if infoCollectionOptions.Timeout > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", infoCollectionOptions.Timeout))
	}

	if infoCollectionOptions.Services {
		args = append(args, "--services")
	}
	if infoCollectionOptions.Dal {
		args = append(args, "--dal")
	}
	return args
}

func (nico *InfoCollectionOptions) All() bool {
	return !nico.Services
}

func (app *DalNode) getInfoCollectionOptions(optionsJson []byte) *InfoCollectionOptions {
	result := &InfoCollectionOptions{}
	json.Unmarshal(optionsJson, result)
	return result
}

func (app *DalNode) GetAvailableInfoCollectionOptions() []base.AmiInfoCollectionOption {
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

func (app *DalNode) GetInfoFromOptions(options *InfoCollectionOptions) (Info, error) {
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

func (app *DalNode) GetInfo(optionsJson []byte) (any, error) {
	return app.GetInfoFromOptions(app.getInfoCollectionOptions(optionsJson))
}

func (app *DalNode) GetServiceInfo() (map[string]base.AmiServiceInfo, error) {
	result := map[string]base.AmiServiceInfo{}

	info, err := app.GetInfoFromOptions(&InfoCollectionOptions{Services: true})
	if err != nil {
		return result, err
	}

	return info.Services, err
}

func (app *DalNode) IsServiceStatus(id string, status string) (bool, error) {
	return base.IsServiceStatus(app, id, status)
}

func (app *DalNode) IsAnyServiceStatus(status string) (bool, error) {
	return base.IsAnyServiceStatus(app, status)
}

func (app *DalNode) PrintInfo(optionsJson []byte) error {
	dalInfoRaw, err := app.GetInfo(optionsJson)
	if err != nil {
		return err
	}
	dalInfo, ok := dalInfoRaw.(Info)
	if !ok {
		return fmt.Errorf("invalid tezpay info type")
	}

	dalTable := table.NewWriter()
	dalTable.SetStyle(table.StyleLight)
	dalTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	dalTable.SetOutputMirror(os.Stdout)
	dalTable.AppendHeader(table.Row{app.GetLabel(), app.GetLabel()}, table.RowConfig{AutoMerge: true})

	dalTable.AppendRow(table.Row{"Status", dalInfo.Status})
	dalTable.AppendRow(table.Row{"Status Level", dalInfo.Level})

	dalTable.AppendSeparator()
	dalTable.AppendRow(table.Row{"Attester Profiles", "Attester Profiles"}, table.RowConfig{AutoMerge: true})
	dalTable.AppendSeparator()
	if len(dalInfo.AttesterProfiles) == 0 {
		dalInfo.AttesterProfiles = []string{"-"}
	}
	for _, profile := range dalInfo.AttesterProfiles {
		dalTable.AppendRow(table.Row{profile, profile}, table.RowConfig{AutoMerge: true})
	}
	dalTable.AppendSeparator()

	dalTable.AppendSeparator()
	dalTable.AppendRow(table.Row{"Services", "Services"}, table.RowConfig{AutoMerge: true})
	dalTable.AppendSeparator()
	dalTable.AppendRow(table.Row{"Name", "Status (Started)"})
	for k, v := range dalInfo.Services {
		dalTable.AppendRow(table.Row{k, fmt.Sprintf("%v (%v)", v.Status, v.Started)})
	}
	dalTable.AppendSeparator()

	dalTable.Render()
	return nil
}
