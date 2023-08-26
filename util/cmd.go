package util

import "github.com/spf13/cobra"

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
