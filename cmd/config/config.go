package configcmd

import (
	"fmt"
	"strings"

	internalconfig "github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var printer *cprint.CPrinter

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Hyphen CLI configuration",
	Long:  `Manage global Hyphen CLI configuration stored on disk.`,
	Args:  cobra.NoArgs,
}

var AutoUpdateCmd = &cobra.Command{
	Use:   "auto-update <on|off>",
	Short: "Enable or disable automatic CLI updates",
	Long:  `Enable or disable automatic CLI updates by writing a setting to your global .hx config.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)

		enabled, err := parseAutoUpdateEnabledArg(args[0])
		if err != nil {
			return err
		}

		if err := internalconfig.UpsertGlobalAutoUpdateEnabled(enabled); err != nil {
			return fmt.Errorf("failed to update auto-update setting: %w", err)
		}

		if enabled {
			printer.Success("Automatic CLI updates are enabled.")
			return nil
		}

		printer.Success("Automatic CLI updates are disabled.")
		return nil
	},
}

func parseAutoUpdateEnabledArg(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "enable", "enabled", "true", "1", "yes", "y":
		return true, nil
	case "off", "disable", "disabled", "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value %q. Expected one of: on, off", value)
	}
}

func init() {
	ConfigCmd.AddCommand(AutoUpdateCmd)
}
