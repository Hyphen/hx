package appcloud

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Hyphen/cli/internal/appcloud"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

// newConfigCmd builds a `config` command with `get`/`set` subcommands. When
// `useDefaultApp` is true the app id comes from the selected default app
// (`hx appcloud app <id>`); otherwise it's a leading positional argument.
func newConfigCmd(useDefaultApp bool) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Read or replace an app's serving config",
	}

	getArgs, setArgs := cobra.ExactArgs(1), cobra.ExactArgs(2)
	getUse, setUse := "get <app-id>", "set <app-id> <file>"
	if useDefaultApp {
		getArgs, setArgs = cobra.NoArgs, cobra.ExactArgs(1)
		getUse, setUse = "get", "set <file>"
	}

	getCmd := &cobra.Command{
		Use:   getUse,
		Short: "Print the app's config as JSON",
		Args:  getArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			appID, err := configTargetApp(useDefaultApp, args, false)
			if err != nil {
				return err
			}
			cv, err := newAppCloudService().GetConfig(appID)
			if err != nil {
				return err
			}
			out, err := json.MarshalIndent(cv.Config, "", "  ")
			if err != nil {
				return errors.Wrap(err, "render config")
			}
			fmt.Println(string(out))
			return nil
		},
	}

	var ifMatch string
	setCmd := &cobra.Command{
		Use:   setUse,
		Short: "Replace the app's config from a JSON file (use - for stdin)",
		Args:  setArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := cprint.NewCPrinter(flags.VerboseFlag)
			appID, err := configTargetApp(useDefaultApp, args, true)
			if err != nil {
				return err
			}
			file := args[len(args)-1]
			raw, err := readFileOrStdin(file)
			if err != nil {
				return err
			}
			var cfg appcloud.AppConfig
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return errors.Wrap(err, "parse config JSON")
			}
			if _, err := newAppCloudService().SetConfig(appID, cfg, ifMatch); err != nil {
				return err
			}
			printer.Success("Updated config for app " + appID)
			return nil
		},
	}
	setCmd.Flags().StringVar(&ifMatch, "if-match", "", "Config etag for optimistic concurrency (rejects with 412 on mismatch)")

	configCmd.AddCommand(getCmd, setCmd)
	return configCmd
}

// configTargetApp resolves the app id for a config subcommand given whether it
// uses the default app and whether the command also takes a trailing file arg.
func configTargetApp(useDefaultApp bool, args []string, hasFileArg bool) (string, error) {
	if useDefaultApp {
		return resolveAppID("")
	}
	// Positional app id is the first argument in both get (<app-id>) and
	// set (<app-id> <file>) forms.
	_ = hasFileArg
	return args[0], nil
}

func readFileOrStdin(path string) ([]byte, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, errors.Wrap(err, "read config from stdin")
		}
		return b, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "read %s", path)
	}
	return b, nil
}
