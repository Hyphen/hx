package autoinit

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Hyphen/cli/cmd/initapp"
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

const guidance = "Hyphen app config is missing or incomplete. Run `hx init` in this directory, then rerun this command."

var (
	isStdinTerminal     = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	runInitApp          = initapp.RunInitAppE
	ensureAuthenticated = user.ErrorIfNotAuthenticated
)

func Ensure(cmd *cobra.Command, args []string) error {
	if !NeedsAppConfig(cmd, args) {
		return nil
	}

	complete, err := HasCompleteAppConfig()
	if err != nil {
		return fmt.Errorf("failed to read .hx config: %w", err)
	}
	if complete {
		return nil
	}

	if shouldFailWithoutPrompt(cmd) {
		return fmt.Errorf("%s", guidance)
	}

	if err := ensureAuthenticated(); err != nil {
		return err
	}

	if err := runInitApp(cmd, []string{}); err != nil {
		return fmt.Errorf("auto-init failed: %w", err)
	}

	complete, err = HasCompleteAppConfig()
	if err != nil {
		return fmt.Errorf("auto-init did not create a readable .hx config: %w", err)
	}
	if !complete {
		return fmt.Errorf("auto-init did not produce a complete app config. %s", guidance)
	}

	cprint.NewCPrinter(flags.VerboseFlag).Info(fmt.Sprintf("Local Hyphen app initialized. Continuing with `%s`.", cmd.CommandPath()))
	return nil
}

// HasCompleteAppConfig reports whether the merged global+local .hx config
// contains every field required by the guarded commands. This mirrors what
// those commands themselves see via RestoreConfigFromFile, so the guard does
// not reject setups the command would accept.
func HasCompleteAppConfig() (bool, error) {
	cfg, found, err := config.RestoreMergedConfig(config.ManifestConfigFile)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	return strings.TrimSpace(cfg.OrganizationId) != "" &&
		nonEmptyStringPtr(cfg.ProjectId) &&
		nonEmptyStringPtr(cfg.AppId) &&
		nonEmptyStringPtr(cfg.AppAlternateId), nil
}

func NeedsAppConfig(cmd *cobra.Command, args []string) bool {
	names := commandNames(cmd)
	if len(names) == 0 {
		return false
	}

	if matches(names, "build") ||
		matches(names, "pull") ||
		matches(names, "push") ||
		matches(names, "env", "pull") ||
		matches(names, "env", "push") ||
		matches(names, "env", "list") ||
		matches(names, "env", "list-versions") ||
		matches(names, "env", "run") ||
		matches(names, "env", "rotate-key") ||
		matches(names, "code", "docker") {
		return true
	}

	if matches(names, "deploy") {
		return deployNeedsAppConfig(cmd, args)
	}

	return false
}

func deployNeedsAppConfig(cmd *cobra.Command, args []string) bool {
	if len(args) == 0 {
		return true
	}

	if stringFlag(cmd, "apps") != "" {
		return false
	}

	return !boolFlag(cmd, "no-build")
}

func shouldFailWithoutPrompt(cmd *cobra.Command) bool {
	return boolFlag(cmd, "no") || isJSONOutput(cmd) || (!isStdinTerminal() && !boolFlag(cmd, "yes"))
}

func isJSONOutput(cmd *cobra.Command) bool {
	return strings.EqualFold(stringFlag(cmd, "output"), cprint.FormatJSON)
}

func commandNames(cmd *cobra.Command) []string {
	var names []string
	for current := cmd; current != nil && current.Parent() != nil; current = current.Parent() {
		names = append([]string{current.Name()}, names...)
	}
	return names
}

func matches(actual []string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range expected {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}

func nonEmptyStringPtr(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

func stringFlag(cmd *cobra.Command, name string) string {
	flag := lookupFlag(cmd, name)
	if flag == nil {
		return ""
	}
	return strings.TrimSpace(flag.Value.String())
}

func boolFlag(cmd *cobra.Command, name string) bool {
	flag := lookupFlag(cmd, name)
	if flag == nil {
		return false
	}
	value, err := strconv.ParseBool(flag.Value.String())
	if err != nil {
		return false
	}
	return value
}

func lookupFlag(cmd *cobra.Command, name string) *pflag.Flag {
	if cmd == nil {
		return nil
	}
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag
	}
	if flag := cmd.InheritedFlags().Lookup(name); flag != nil {
		return flag
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		return flag
	}
	return nil
}
