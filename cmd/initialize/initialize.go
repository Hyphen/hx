package initialize

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/cmd/initapp"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
)

var appIDFlag string
var isMonorepo bool
var printer *cprint.CPrinter

var InitCmd = &cobra.Command{
	Use:   "init <app name>",
	Short: "Initialize a new Hyphen application or monorepo project in the current directory",
	Long: `
The init command sets up a new Hyphen application or monorepo project in your current directory.

This command will:
- Create a new application or project in Hyphen
- Generate a local configuration file
- Set up environment files for each project environment
- Update .gitignore to exclude sensitive files

If no app name is provided, it will prompt to use the current directory name.

For single applications:
The command will guide you through:
- Confirming or entering an application name
- Generating or providing an app ID
- Creating necessary local files

For monorepos (using --monorepo flag):
The command will guide you through:
- Setting up a project structure
- Adding multiple applications within the project
- Configuring each application with its own settings
- Creating environment files for each application

After initialization, you'll receive a summary of the new application(s), including name(s), 
ID(s), and associated organization.

Examples:
  hyphen init                                    # Initialize single app
  hyphen init "My New App"                       # Initialize single app with name
  hyphen init "My New App" --id my-custom-app-id # Initialize single app with custom ID
  hyphen init --monorepo                         # Initialize monorepo project
  hyphen init "My Project" --monorepo            # Initialize monorepo with project name
`,
	Args: cobra.MaximumNArgs(1),
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
