package initialize

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/cmd/initapp"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var appIDFlag string
var isMonorepo bool
var printer *cprint.CPrinter

var InitCmd = &cobra.Command{
	Use:   "init <app name>",
	Short: "Initialize a new Hyphen application in the current directory",
	Long: `
The init command sets up a new Hyphen application in your current directory.

This command will:
- Create a new application in Hyphen
- Generate a local configuration file
- Set up environment files for each project environment
- Update .gitignore to exclude sensitive files

If no app name is provided, it will prompt to use the current directory name.

The command will guide you through:
- Confirming or entering an application name
- Generating or providing an app ID
- Creating necessary local files

After initialization, you'll receive a summary of the new application, including its name, 
ID, and associated organization.

Examples:
  hyphen init
  hyphen init "My New App"
  hyphen init "My New App" --id my-custom-app-id
`,
	Args: cobra.MaximumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if isMonorepo {
			runInitMonorepo(cmd, args)
		} else {
			initapp.RunInitApp(cmd, args)

		}
	},
}

func init() {
	InitCmd.Flags().StringVarP(&appIDFlag, "id", "i", "", "App ID (optional)")
	InitCmd.Flags().BoolVarP(&isMonorepo, "monorepo", "m", false, "Initialize a monorepo")
	InitCmd.Flags().BoolVarP(&flags.LocalSecret, "localSecret", "l", false, "Use local secret key file instead of Hyphen's secure key store")
}

func isValidDirectory(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if cwd == homeDir {
		return fmt.Errorf("initialization in home directory not allowed")
	}

	return nil
}
