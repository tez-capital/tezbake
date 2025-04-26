package util

import (
	"os"
	"slices"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func GetCommandStringFlag(cmd *cobra.Command, flag string) string {
	value, err := cmd.Flags().GetString(flag)
	AssertE(err, "Failed to get command flag!")
	return value
}

func GetCommandStringFlagD(cmd *cobra.Command, flag string, def string) string {
	value, err := cmd.Flags().GetString(flag)
	AssertE(err, "Failed to get command flag!")
	if value == "" {
		return def
	}
	return value
}
func GetCommandBoolFlag(cmd *cobra.Command, flag string) bool {
	value, err := cmd.Flags().GetBool(flag)
	AssertE(err, "Failed to get command flag!")
	return value
}

func GetCommandStringFlagS(cmd *cobra.Command, flag string) string {
	value, _ := cmd.Flags().GetString(flag)
	return value
}

func GetCommandStringFlagSD(cmd *cobra.Command, flag string, def string) string {
	value, _ := cmd.Flags().GetString(flag)
	if value == "" {
		return def
	}
	return value
}

func GetCommandBoolFlagS(cmd *cobra.Command, flag string) bool {
	value, _ := cmd.Flags().GetBool(flag)
	return value
}

// GetCommandArgs retrieves the arguments specified after the command.
// It returns nil if no arguments are provided.
func GetCommandArgs(cmd *cobra.Command) []string {
	args := os.Args
	for i, arg := range os.Args {
		if arg == cmd.Use {
			return args[i+1:]
		}
	}

	return nil
}

func RemoveCmdFlags(cmd *cobra.Command, args []string) []string {
	flags := []string{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, "--"+f.Name)
	})
	return lo.Filter(args, func(arg string, _ int) bool {
		return !slices.Contains(flags, arg)
	})
}
