package signer

import (
	"fmt"
	"strings"

	"github.com/tez-capital/tezbake/ami"
)

func (app *Signer) GetKeyHash(alias string, args ...string) (string, int, error) {
	execArgs := make([]string, 0)
	execArgs = append(execArgs, "get-key-hash")
	execArgs = append(execArgs, fmt.Sprintf("--alias=%s", alias))
	execArgs = append(execArgs, args...)
	output, exitCode, err := ami.ExecuteGetOutput(app.GetPath(), execArgs...)
	if err != nil {
		return output, exitCode, fmt.Errorf("failed to get key hash from signer (%s)", err.Error())
	}
	trimmedOutput := strings.TrimSpace(output)
	lastLine := trimmedOutput[strings.LastIndex(trimmedOutput, "\n")+1:]
	key := strings.TrimSpace(lastLine)
	return key, exitCode, nil
}
