package run

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var printer *cprint.CPrinter

var RunCmd = &cobra.Command{
	Use:   "run [environment] -- [command]",
	Short: "Run your app with the specified environment",
	Long: `
The run command executes your application with the specified environment variables.

Usage:
  hyphen env run [environment] -- [command]

Examples:
  hyphen env run production -- go run main.go
  hyphen env run staging -- node server.js
  hyphen env run -- go run main.go (uses default environment)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)

		var envName string
		var childCommand []string

		separatorIndex := cmd.ArgsLenAtDash()

		if separatorIndex == -1 || separatorIndex == 0 {
			// No environment specified or "--" is the first argument
			envName = "default"
			childCommand = args
		} else {
			envName = args[0]
			childCommand = args[separatorIndex:]
		}

		if len(childCommand) == 0 {
			return errors.New("No command specified")
		}

		config, err := config.RestoreConfig()
		if err != nil {
			return errors.Wrap(err, "Failed to restore manifest")
		}

		mergedEnvVars, err := loadAndMergeEnvFiles(envName, config)
		if err != nil {
			return err
		}

		if err := runCommandWithEnv(childCommand, mergedEnvVars); err != nil {
			return errors.Wrap(err, "Command execution failed")
		}
		return nil
	},
}

func loadAndMergeEnvFiles(envName string, config config.Config) ([]string, error) {
	var mergedVars []string

	// Load .env file (default)
	if err := loadAndAppendEnv("default", config, &mergedVars); err != nil && envName == "default" {
		return nil, err // Return error if default is specifically requested and doesn't exist
	}

	// Load .env.local file (if exists)
	if err := loadAndAppendEnv("local", config, &mergedVars); err != nil && envName == "local" {
		return nil, err // Return error if local is specifically requested and doesn't exist
	}

	// Load .env.<environment> file (if provided)
	if envName != "default" && envName != "local" {
		if err := loadAndAppendEnv(envName, config, &mergedVars); err != nil {
			return nil, err
		}
	}

	return mergedVars, nil
}

func loadAndAppendEnv(envName string, config config.Config, mergedVars *[]string) error {
	envContents, err := env.GetLocalEnvContents(envName)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Wrap(err, fmt.Sprintf("%s env file not found", envName))
		}
		return errors.Wrap(err, fmt.Sprintf("Error loading %s env file", envName))
	}

	scanner := bufio.NewScanner(strings.NewReader(envContents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if env.IsEnvVar(line) {
			*mergedVars = append(*mergedVars, line)
		}
	}

	if flags.VerboseFlag {
		printer.Info(fmt.Sprintf("Loaded and appended %s environment", envName))
	}
	return nil
}

func runCommandWithEnv(command []string, envVars []string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(os.Environ(), envVars...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if flags.VerboseFlag {
		printer.Info(fmt.Sprintf("Running command with %s environment", command[0]))
		printer.Info(fmt.Sprintf("Executing command: %s", strings.Join(command, " ")))
	}
	return cmd.Run()
}
