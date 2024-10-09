package run

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"dario.cat/mergo"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

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
`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		envName := args[0]
		childCommand := args[1:]

		// Find the index of "--" if it exists
		separatorIndex := cmd.ArgsLenAtDash()
		if separatorIndex != -1 {
			childCommand = args[separatorIndex:]
		}

		manifest, err := manifest.Restore()
		if err != nil {
			return errors.Wrap(err, "Failed to restore manifest")
		}

		// Load and merge env files
		mergedEnvVars, err := loadAndMergeEnvFiles(envName, manifest)
		if err != nil {
			return err
		}

		// Run the child command with the merged env vars
		if err := runCommandWithEnv(childCommand, mergedEnvVars); err != nil {
			cprint.Error(cmd, errors.Wrap(err, "Command execution failed"))
			return err
		}

		return nil
	},
}

func loadAndMergeEnvFiles(envName string, m manifest.Manifest) ([]string, error) {
	mergedVars := make(map[string]string)

	// Load .env file (default)
	if err := loadAndMergeEnv("default", m, mergedVars, false); err != nil {
		return nil, err
	}

	// Load .env.<environment> file
	if err := loadAndMergeEnv(envName, m, mergedVars, true); err != nil {
		return nil, err
	}

	// Convert merged map to slice of strings
	var result []string
	for k, v := range mergedVars {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result, nil
}

func loadAndMergeEnv(envName string, m manifest.Manifest, mergedVars map[string]string, override bool) error {
	envFile, err := env.GetLocalEnv(envName, m)
	if err != nil {
		if os.IsNotExist(err) && envName == "default" {
			// It's okay if the default .env file doesn't exist
			return nil
		}
		return errors.Wrap(err, fmt.Sprintf("Error loading %s env file", envName))
	}

	decrypted, err := envFile.DecryptData(m.GetSecretKey())
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error decrypting %s env", envName))
	}

	vars := parseEnvString(decrypted)
	mergeOpt := mergo.WithAppendSlice
	if override {
		mergeOpt = mergo.WithOverride
	}
	if err := mergo.Merge(&mergedVars, vars, mergeOpt); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error merging %s variables", envName))
	}

	if flags.VerboseFlag {
		cprint.Info(fmt.Sprintf("Loaded and merged %s environment", envName))
	}
	return nil
}

func parseEnvString(envData string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(envData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if env.IsEnvVar(line) {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				result[key] = value
			}
		}
	}
	return result
}

func runCommandWithEnv(command []string, envVars []string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(os.Environ(), envVars...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if flags.VerboseFlag {
		cprint.Info(fmt.Sprintf("Running command with %s environment", command[0]))
		cprint.Info(fmt.Sprintf("Executing command: %s", strings.Join(command, " ")))
	}
	return cmd.Run()
}
