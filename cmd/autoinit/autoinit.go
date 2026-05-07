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

const guidance = "local Hyphen app config is missing or incomplete. Run `hx init` in this directory, then rerun this command."

var (
	isStdinTerminal     = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	runInitApp          = initapp.RunInitAppE
	ensureAuthenticated = user.ErrorIfNotAuthenticated
)

func Ensure(cmd *cobra.Command, args []string) error {
	if !NeedsLocalAppConfig(cmd, args) {
		return nil
	}

	complete, err := HasCompleteLocalAppConfig()
	if err != nil {
		return fmt.Errorf("failed to read local .hx config: %w", err)
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

	complete, err = HasCompleteLocalAppConfig()
	if err != nil {
		return fmt.Errorf("auto-init did not create a readable local .hx config: %w", err)
	}
	if !complete {
		return fmt.Errorf("auto-init did not create complete local app config. %s", guidance)
	}

	cprint.NewCPrinter(flags.VerboseFlag).Info(fmt.Sprintf("Local Hyphen app initialized. Continuing with `%s`.", cmd.CommandPath()))
	return nil
}

func HasCompleteLocalAppConfig() (bool, error) {
	cfg, err := config.RestoreLocalConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return strings.TrimSpace(cfg.OrganizationId) != "" &&
		nonEmptyStringPtr(cfg.ProjectId) &&
		nonEmptyStringPtr(cfg.AppId) &&
		nonEmptyStringPtr(cfg.AppAlternateId), nil
}

func NeedsLocalAppConfig(cmd *cobra.Command, args []string) bool {
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
		return deployNeedsLocalAppConfig(cmd, args)
	}

	return false
}

func deployNeedsLocalAppConfig(cmd *cobra.Command, args []string) bool {
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
