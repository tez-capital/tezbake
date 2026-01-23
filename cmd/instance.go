package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
)

// enterInstanceEnvironment spawns an interactive shell with the instance environment
func enterInstanceEnvironment(alias, instancePath string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// Create the instance path if it doesn't exist
	// If creation fails (likely permission denied), try with sudo for just the mkdir
	if err := os.MkdirAll(instancePath, 0755); err != nil {
		if os.IsPermission(err) {
			// Only elevate the mkdir operation, not the whole command
			// This way the user enters the shell as themselves, not root
			mkdirCmd := exec.Command("sudo", "mkdir", "-p", instancePath)
			mkdirCmd.Stdin = os.Stdin
			mkdirCmd.Stdout = os.Stdout
			mkdirCmd.Stderr = os.Stderr
			if err := mkdirCmd.Run(); err != nil {
				return fmt.Errorf("failed to create instance path with sudo: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create instance path: %w", err)
		}
	}

	// Set up the environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("TEZBAKE_INSTANCE_PATH=%s", instancePath))

	// Set custom colorful PS1 based on shell type and determine shell args
	// Format: tezbake ❯ instance ❯ <alias> <pwd> ❯
	// Colors based on tez.capital branding with powerline-style arrows
	shellName := filepath.Base(shell)
	var shellArgs []string

	switch shellName {
	case "zsh":
		// Zsh uses %F{color} for colors, %B %b for bold, %~ for pwd with ~ substitution
		// Use -f to prevent loading .zshrc which would override PS1
		ps1 := fmt.Sprintf("%%F{cyan}%%Btezbake%%b%%f %%F{blue}❯%%f %%F{75}instance%%f %%F{blue}❯%%f %%F{159}%%B%s%%b%%f %%F{243}%%~%%f %%F{blue}❯%%f ", alias)
		env = append(env, "PS1="+ps1)
		shellArgs = append(shellArgs, "-f")
	case "bash":
		// Bash uses \[ \] to wrap non-printing characters, \w for current directory
		// Use --norc --noprofile to prevent loading config files
		ps1 := fmt.Sprintf("\\[\\033[1;36m\\]tezbake\\[\\033[0m\\] \\[\\033[34m\\]❯\\[\\033[0m\\] \\[\\033[94m\\]instance\\[\\033[0m\\] \\[\\033[34m\\]❯\\[\\033[0m\\] \\[\\033[1;96m\\]%s\\[\\033[0m\\] \\[\\033[90m\\]\\w\\[\\033[0m\\] \\[\\033[34m\\]❯\\[\\033[0m\\] ", alias)
		env = append(env, "PS1="+ps1)
		shellArgs = append(shellArgs, "--norc", "--noprofile")
	default:
		// For other shells, use simple ANSI codes with $PWD
		ps1 := fmt.Sprintf("\033[1;36mtezbake\033[0m \033[34m❯\033[0m \033[94minstance\033[0m \033[34m❯\033[0m \033[1;96m%s\033[0m \033[90m$PWD\033[0m \033[34m❯\033[0m ", alias)
		env = append(env, "PS1="+ps1)
	}

	// Create the command with appropriate flags
	shellCmd := exec.Command(shell, shellArgs...)
	shellCmd.Env = env
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	shellCmd.Dir = instancePath

	err := shellCmd.Run()
	// Exit with the shell's exit code to prevent cobra from printing help
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
	return nil // unreachable, but required by compiler
}

var instanceCmd = &cobra.Command{
	Use:   "instance [alias] [command]",
	Short: "Executes command on a specific tezbake instance or enters the instance environment",
	Long: `Proxies the command to the specified tezbake instance by setting the appropriate path.
If no command is provided, enters an interactive shell environment for the instance.`,
	Aliases:            []string{"i"},
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Manually handle help since DisableFlagParsing is true
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				return cmd.Help()
			}
		}

		if len(args) < 1 {
			return fmt.Errorf("instance alias is required")
		}

		if args[0] == "list" {
			return listCmd.RunE(cmd, args)
		}

		alias := args[0]
		// Determine the new path for the instance
		var instancePath string
		if alias == "default" || strings.ToLower(alias) == "default" {
			instancePath = constants.DefaultBBDirectory
		} else {
			instancePath = filepath.Join(constants.DefaultBBDirectory, "instances", alias)
		}

		// If no additional args, enter the instance environment
		if len(args) == 1 {
			return enterInstanceEnvironment(alias, instancePath)
		}

		// Prepare arguments for the recursive execution
		// We prepend --path <instancePath> to the arguments
		newArgs := append([]string{"--path", instancePath}, args[1:]...)

		// Reset the RootCmd args and execute it again with the new arguments
		RootCmd.SetArgs(newArgs)
		return RootCmd.Execute()
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available instances",
	Long:  "Lists all available tezbake instances found in the default directory.",
	RunE: func(cmd *cobra.Command, args []string) error {
		bbDir := cli.BBdir
		if bbDir == "" {
			bbDir = constants.DefaultBBDirectory
		}

		instances := []string{}

		// Check for default instance
		// Default instance exists if there is any other directory in the /bake-buddy except instances
		entries, err := os.ReadDir(bbDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && entry.Name() != "instances" {
					instances = append(instances, "default")
					break
				}
			}
		}

		// Check for other instances
		instancesDir := filepath.Join(bbDir, "instances")
		entries, err = os.ReadDir(instancesDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					instances = append(instances, entry.Name())
				}
			}
		}

		if cli.JsonLogFormat {
			output := make([]map[string]string, 0, len(instances))
			for _, instance := range instances {
				location := filepath.Join(bbDir, "instances", instance)
				if instance == "default" {
					location = bbDir
				}
				output = append(output, map[string]string{"name": instance, "location": location})
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
		}

		for _, instance := range instances {
			location := filepath.Join(bbDir, "instances", instance)
			if instance == "default" {
				location = bbDir
			}
			fmt.Fprintf(cmd.OutOrStdout(), "- %s\t\t%s\n", instance, location)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(instanceCmd)
	instanceCmd.AddCommand(listCmd)
}
