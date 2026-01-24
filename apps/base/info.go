package base

import (
	"bytes"
	"encoding/json"
	"strings"
)

type InfoBase struct {
	Level  string `json:"level"`
	Status string `json:"status"`
	Output string `json:"output"`
}

func GenerateFailedInfo(output string, err error) InfoBase {
	return InfoBase{
		Level:  "error",
		Status: "Failed to get app status - '" + err.Error() + "!",
		Output: output,
	}
}

func ParseInfoOutput[TInfo any](infoBytes []byte) (TInfo, error) {
	info := string(infoBytes)
	lines := strings.Split(info, "\n")
	i := len(lines) - 1
	for strings.TrimSpace(lines[i]) == "" {
		i--
	}
	info = lines[i]

	var result TInfo
	err := json.Unmarshal([]byte(info), &result)
	return result, err
}

func isEmptyArray(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return bytes.Equal(trimmed, []byte("[]"))
}

func UnmarshalIfNotEmptyArray[T any](data json.RawMessage, result T) error {
	if len(data) == 0 || isEmptyArray(data) {
		return nil
	}
	return json.Unmarshal(data, result)
}

type AppServiceStatusCollector interface {
	GetServiceInfo() (map[string]AmiServiceInfo, error)
}

func IsServiceStatus(app AppServiceStatusCollector, id string, status string) (bool, error) {
	serviceInfo, err := app.GetServiceInfo()
	if err != nil {
		return false, err
	}
	if service, ok := serviceInfo[id]; ok && service.Status == status {
		return true, nil
	}
	return false, nil
}

func IsAnyServiceStatus(app AppServiceStatusCollector, status string) (bool, error) {
	serviceInfo, err := app.GetServiceInfo()
	if err != nil {
		return false, err
	}
	for _, service := range serviceInfo {
		if service.Status == status {
			return true, nil
		}
	}
	return false, nil
}
