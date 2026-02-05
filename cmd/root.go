package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"go.alis.is/common/log"
)

const (
	LOG_LEVEL_FLAG               = "log-level"
	NO_COLOR_FLAG                = "no-color"
	PATH_FLAG                    = "path"
	REMOTE_INSTANCE_VARS_FLAG    = "remote-instance-vars"
	VERSION_FLAG                 = "version"
	DISABLE_DONATION_PROMPT_FLAG = "disable-donation-prompt"
	OUTPUT_FORMAT_FLAG           = "output-format"
	PAY_ONLY_ADDRESS_PREFIX      = "pay-only-address-prefix"
)

func setupLogger(level slog.Level, format string, noColor bool) (jsonLogFormat bool) {
	var jsonWriters []io.Writer

	textWriters := []io.Writer{os.Stdout}

	switch format {
	case "json":
		jsonWriters = append(jsonWriters, os.Stdout)
		jsonLogFormat = true
	case "text":
		textWriters = append(textWriters, os.Stdout)
		jsonLogFormat = false
	}

	handlers := make([]slog.Handler, 0, 2)
	if len(textWriters) > 0 {
		textHandler := log.NewPrettyTextLogHandler(log.NewMultiWriter(textWriters...), log.PrettyHandlerOptions{
			HandlerOptions: slog.HandlerOptions{Level: level},
			NoColor:        noColor,
			AppName:        "tezbake",
		})
		handlers = append(handlers, textHandler)
	}

	if len(jsonWriters) > 0 {
		jsonHandler := slog.NewJSONHandler(log.NewMultiWriter(jsonWriters...), &slog.HandlerOptions{Level: level})
		handlers = append(handlers, jsonHandler)
	}

	slog.SetDefault(slog.New(log.NewSlogMultiHandler(handlers...)))

	return
}

var (
	RootCmd = &cobra.Command{
		Use:   "tezbake",
		Short: "tezbake CLI",
		Long: fmt.Sprintf(`tezbake CLI
Copyright © %d tez.capital
`, time.Now().Year()),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed(PATH_FLAG) {
				cli.BBdir, _ = cmd.Flags().GetString(PATH_FLAG)
			} else {
				bbDir := os.Getenv("TEZBAKE_INSTANCE_PATH")
				if bbDir != "" {
					cli.BBdir = bbDir
				}
			}

			logLevel := slog.LevelInfo
			logLevelFlag, _ := cmd.Flags().GetString(LOG_LEVEL_FLAG)
			logLevel, cli.LogLevel = log.ParseLevel(logLevelFlag)
			format, _ := cmd.Flags().GetString(OUTPUT_FORMAT_FLAG)
			noColor, _ := cmd.Flags().GetBool(NO_COLOR_FLAG)
			cli.JsonLogFormat = setupLogger(logLevel, format, noColor)
			log.Debug("logger configured", "format", format, "level", logLevelFlag)

			remoteVars, _ := cmd.Flags().GetString(REMOTE_INSTANCE_VARS_FLAG)
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
	RootCmd.PersistentFlags().StringP(PATH_FLAG, "p", constants.DefaultBBDirectory, "Path to bake buddy instance")
	RootCmd.PersistentFlags().StringP(OUTPUT_FORMAT_FLAG, "o", "auto", "Sets output log format (json/text/auto)")
	RootCmd.PersistentFlags().StringP(LOG_LEVEL_FLAG, "l", "info", "Sets log level (trace/debug/info/warn/error)")
	RootCmd.PersistentFlags().Bool(NO_COLOR_FLAG, false, "Disable color output")
	RootCmd.PersistentFlags().Bool(VERSION_FLAG, false, "Prints tezbake version")
	defaultHelpFunc := RootCmd.HelpFunc()
	RootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		v, _ := cmd.Flags().GetBool(VERSION_FLAG)
		if v {
			fmt.Println(constants.VERSION)
			os.Exit(0)
		}
		defaultHelpFunc(cmd, args)
	})
	RootCmd.PersistentFlags().String(REMOTE_INSTANCE_VARS_FLAG, "", "Tells tezbake to which remote vars to set (available only with remote-instance)")
	RootCmd.PersistentFlags().MarkHidden(REMOTE_INSTANCE_VARS_FLAG)
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
