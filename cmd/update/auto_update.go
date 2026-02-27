package update

import (
	"fmt"
	"os"
	"strings"

	cliVersion "github.com/Hyphen/cli/cmd/version"
	internalconfig "github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var restoreGlobalConfig = internalconfig.RestoreGlobalConfig

func RunAutoUpdate(cmd *cobra.Command) {
	if shouldSkipAutoUpdateForCommand(cmd) {
		return
	}

	if isCIEnvironment() {
		return
	}

	isEnabled, err := isAutoUpdateEnabledFromConfig()
	if err != nil {
		if flags.VerboseFlag {
			cprint.Warning(fmt.Sprintf("Unable to read auto-update configuration: %v", err))
		}
		return
	}
	if !isEnabled {
		return
	}

	currentVersion := strings.TrimSpace(cliVersion.GetVersion())
	if currentVersion == "" || currentVersion == "unknown" {
		return
	}

	printer = cprint.NewCPrinter(flags.VerboseFlag)
	updater := NewDefaultUpdater("")
	if err := updater.RunAuto(cmd); err != nil {
		if flags.VerboseFlag {
			printer.Warning(fmt.Sprintf("Automatic update check failed: %v", err))
		}
	}
}

func shouldSkipAutoUpdateForCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return shouldSkipAutoUpdateCommandPath(cmd.CommandPath())
}

func shouldSkipAutoUpdateCommandPath(commandPath string) bool {
	pathParts := strings.Fields(strings.ToLower(strings.TrimSpace(commandPath)))
	if len(pathParts) <= 1 {
		return true
	}

	subcommand := pathParts[1]
	switch subcommand {
	case "help", "completion", "update", "version", "config":
		return true
	default:
		return false
	}
}

func isCIEnvironment() bool {
	ciEnvVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"BUILDKITE",
		"CIRCLECI",
		"GITLAB_CI",
		"JENKINS_URL",
		"TF_BUILD",
		"TEAMCITY_VERSION",
		"CODEBUILD_BUILD_ID",
	}

	for _, key := range ciEnvVars {
		if isTruthyEnvValue(os.Getenv(key)) {
			return true
		}
	}

	return false
}

func isTruthyEnvValue(value string) bool {
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	if normalizedValue == "" {
		return false
	}

	switch normalizedValue {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func isAutoUpdateEnabledFromConfig() (bool, error) {
	cfg, err := restoreGlobalConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	return cfg.IsAutoUpdateEnabled(), nil
}
