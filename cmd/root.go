package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/logging"
)

var (
	RootCmd = &cobra.Command{
		Use:   "tezbake",
		Short: "tezbake CLI",
		Long: fmt.Sprintf(`tezbake CLI
Copyright © %d tez.capital
`, time.Now().Year()),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("path") {
				cli.BBdir, _ = cmd.Flags().GetString("path")
			} else {
				bbDir := os.Getenv("TEZBAKE_INSTANCE_PATH")
				if bbDir != "" {
					cli.BBdir = bbDir
				}
			}

			level := slog.LevelInfo
			levelFlag, _ := cmd.Flags().GetString("log-level")
			level, cli.LogLevel = logging.ParseLevel(levelFlag)

			remoteVars, _ := cmd.Flags().GetString("remote-instance-vars")
			if remoteVars != "" {
				vars := strings.Split(remoteVars, ";")
				for _, _var := range vars {
					kvPair := strings.Split(_var, "=")
					if len(kvPair) != 2 {
						continue
					}
					ami.REMOTE_VARS[kvPair[0]] = kvPair[1]
				}
			}

			format, _ := cmd.Flags().GetString("output-format")
			handler, jsonFormat, formatLogMsg := logging.NewHandler(format, level, os.Stdout, os.Stderr)
			cli.JsonLogFormat = jsonFormat

			slog.SetDefault(slog.New(handler))
			logging.Tracef("Log level set to '%s'", cli.LogLevel)
			logging.Trace(formatLogMsg)

			// init ami options
			ami.SetOptions(ami.Options{
				LogLevel:      cli.LogLevel,
				JsonLogFormat: cli.JsonLogFormat,
			})

		},
	}
)

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.PersistentFlags().StringP("path", "p", constants.DefaultBBDirectory, "Path to bake buddy instance")
	RootCmd.PersistentFlags().StringP("output-format", "o", "auto", "Sets output log format (json/text/auto)")
	RootCmd.PersistentFlags().StringP("log-level", "l", "info", "Sets log level (trace/debug/info/warn/error)")
	RootCmd.PersistentFlags().Bool("version", false, "Prints tezbake version")
	defaultHelpFunc := RootCmd.HelpFunc()
	RootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		v, _ := cmd.Flags().GetBool("version")
		if v {
			fmt.Println(constants.VERSION)
			os.Exit(0)
		}
		defaultHelpFunc(cmd, args)
	})
	RootCmd.PersistentFlags().String("remote-instance-vars", "", "Tells tezbake to which remote vars to set (available only with remote-instance)")
	RootCmd.PersistentFlags().MarkHidden("remote-instance-vars")
	RootCmd.PersistentFlags().SetInterspersed(false)
}

func ExecuteTest(t *testing.T, c *cobra.Command, args ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)

	err := c.Execute()
	return strings.TrimSpace(buf.String()), err
}
