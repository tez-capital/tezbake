package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/cli"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type bbTextFormatter struct {
	log.TextFormatter
}

func (f *bbTextFormatter) Format(entry *log.Entry) ([]byte, error) {
	result := entry.Time.Format("15:04:05")
	result = result + " [" + strings.ToUpper(string(entry.Level.String())) + "] (bb-cli) " + entry.Message + "\n"
	for k, v := range entry.Data {
		result = result + k + "=" + fmt.Sprint(v) + "\n"
	}
	return []byte(result), nil
}

type bbJsonFormatter struct {
	log.JSONFormatter
}

func (f *bbJsonFormatter) Format(entry *log.Entry) ([]byte, error) {
	//strconv.FormatInt(entry.Time.Unix(), 10)
	l, err := f.JSONFormatter.Format(entry)
	if err != nil {
		return []byte{}, err
	}
	result := make(map[string]interface{})
	err = json.Unmarshal(l, &result)
	if err != nil {
		return []byte{}, err
	}
	delete(result, "time")
	result["timestamp"] = strconv.FormatInt(entry.Time.Unix(), 10)
	result["module"] = "bb-cli"
	resultLog, err := json.Marshal(result)
	resultLog = append(resultLog, byte('\n'))
	return resultLog, err
}

var (
	RootCmd = &cobra.Command{
		Use:   "bb-cli",
		Short: "Bake Buddy CLI",
		Long: `Bake Buddy CLI
Copyright © 2022 alis.is & GrokTech
`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {

			if cmd.Flags().Changed("path") {
				cli.BBdir, _ = cmd.Flags().GetString("path")
			} else {
				bbDir := os.Getenv("BB_DIR")
				if bbDir != "" {
					cli.BBdir = bbDir
				}
			}

			switch level, _ := cmd.Flags().GetString("log-level"); level {
			case "trace":
				log.SetLevel(log.TraceLevel)
				cli.LogLevel = "trace"
			case "debug":
				log.SetLevel(log.DebugLevel)
				cli.LogLevel = "debug"
			case "warn":
				log.SetLevel(log.WarnLevel)
				cli.LogLevel = "warn"
			case "error":
				log.SetLevel(log.ErrorLevel)
				cli.LogLevel = "error"
			default:
				log.SetLevel(log.InfoLevel)
			}
			log.Trace("Log level set to '" + cli.LogLevel + "'")

			cli.IsRemoteInstance, _ = cmd.Flags().GetBool("remote-instance")
			if cli.IsRemoteInstance {
				if checkModule, _ := cmd.Flags().GetString("is-module-installed"); checkModule != "" {
					for _, v := range bb.Modules {
						if v.GetId() == checkModule {
							fmt.Println(v.IsInstalled())
							os.Exit(0)
						}
					}
				}

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
			}

			format, _ := cmd.Flags().GetString("output-format")
			// if cli.IsRemoteInstance { // override for remote instance
			// 	format = "json"
			// }

			switch format {
			case "json":
				cli.JsonLogFormat = true
				log.SetFormatter(&bbJsonFormatter{})
				log.Trace("Output format set to 'json'")
			case "text":
				log.SetFormatter(&bbTextFormatter{})
				log.Trace("Output format set to 'text'")
			default:
				if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
					log.SetFormatter(&bbJsonFormatter{})
					log.Trace("Output format automatically set to 'json'")
				} else {
					log.SetFormatter(&bbTextFormatter{})
					log.Trace("Output format automatically set to 'text'")
				}
			}

		},
	}
)

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.PersistentFlags().StringP("path", "p", cli.DefaultBBDirectory, "Path to bake buddy instance")
	RootCmd.PersistentFlags().StringP("output-format", "o", "auto", "Sets output log format (json/text/auto)")
	RootCmd.PersistentFlags().StringP("log-level", "l", "info", "Sets output log format (json/text/auto)")
	RootCmd.PersistentFlags().Bool("remote-instance", false, "Tells bb-cli to operate in remote-instance mode")
	RootCmd.PersistentFlags().MarkHidden("remote-instance")
	RootCmd.PersistentFlags().String("remote-instance-vars", "", "Tells bb-cli to which remote vars to set (available only with remote-instance)")
	RootCmd.PersistentFlags().MarkHidden("remote-instance-vars")
	RootCmd.PersistentFlags().String("is-module-installed", "", "Checks whether module is installed (available only with remote-instance)")
	RootCmd.PersistentFlags().MarkHidden("is-module-installed")
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
