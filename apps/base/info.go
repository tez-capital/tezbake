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

func ParseInfoOutput(infoBytes []byte) (map[string]interface{}, error) {
	info := string(infoBytes)
	lines := strings.Split(info, "\n")
	i := len(lines) - 1
	for strings.TrimSpace(lines[i]) == "" {
		i--
	}
	info = lines[i]

	resultMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(info), &resultMap)
	if err == nil {
		return resultMap, nil
	}
	return map[string]interface{}{
		"level":  "error",
		"status": "Failed to parse ami info - '" + err.Error() + "!",
		"output": info,
	}, err
}
