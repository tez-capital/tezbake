package base

import (
	"encoding/json"
	"strings"
)

func GenerateFailedInfo(output string, err error) map[string]interface{} {
	return map[string]interface{}{
		"level":  "error",
		"status": "Failed to get app status - '" + err.Error() + "!",
		"output": output,
	}
}

func ParseInfoOutput[T any](infoBytes []byte) (map[string]T, error) {
	info := string(infoBytes)
	lines := strings.Split(info, "\n")
	i := len(lines) - 1
	for strings.TrimSpace(lines[i]) == "" {
		i--
	}
	info = lines[i]

	resultMap := make(map[string]T)
	err := json.Unmarshal([]byte(info), &resultMap)
	if err == nil {
		return resultMap, nil
	}

	x := any(*new(T))
	switch x.(type) {
	case json.RawMessage:
		return map[string]T{
			"level":  any(json.RawMessage("error")).(T),
			"status": any(json.RawMessage("Failed to parse ami info - '" + err.Error() + "!")).(T),
			"output": any(json.RawMessage(info)).(T),
		}, err
	default:
		return map[string]T{
			"level":  any("error").(T),
			"status": any("Failed to parse ami info - '" + err.Error() + "!").(T),
			"output": any(info).(T),
		}, err
	}

}
