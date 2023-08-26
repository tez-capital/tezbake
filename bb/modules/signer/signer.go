package bb_module_signer

import (
	"fmt"
	"strings"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/cli"
)

func (app *Signer) GetKeyHash(alias string, args ...string) (string, int, error) {
	if cli.IsRemoteInstance {
		return ami.REMOTE_VARS[ami.BAKER_KEY_HASH_REMOTE_VAR], 0, nil
	}
	execArgs := make([]string, 0)
	execArgs = append(execArgs, "get-key-hash")
	execArgs = append(execArgs, fmt.Sprintf("--alias=%s", alias))
	execArgs = append(execArgs, args...)
	output, exitCode, err := ami.ExecuteGetOutput(app.GetPath(), execArgs...)
	if err != nil {
		return output, exitCode, fmt.Errorf("failed to get key hash from signer (%s)", err.Error())
	}
	return strings.Trim(output, " \n"), exitCode, nil
}
