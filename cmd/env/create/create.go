package create

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

var fileName string

var CreateCmd = &cobra.Command{
	Use:     "create [ENVIRONMENT]",
	Aliases: []string{"c"},
	Short:   "Create an environment file",
	Long: `Creates a new environment file for the specified environment.

Example usage:
  hyphen env create default
  hyphen env create production

The command will create a file named based on the environment, such as 'default.env' or 'production.env'.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize env with an empty string
		env := ""

		// If an environment is provided in args, use it
		if len(args) == 1 {
			env = args[0]
		}

		// Get the default environment file name
		fileName = environment.GetEnvFileByEnvironment(env)

		// Check if the file already exists to avoid overwriting
		if _, err := os.Stat(fileName); err == nil {
			fmt.Printf("File %s already exists. Use a different environment or delete the existing file.\n", fileName)
			os.Exit(1)
		}

		// Example content, this should be replaced with actual content or logic to create the file content.
		content := []byte("# Environment variables\nKEY=Value\n")

		// Write the initial content to the file
		err := os.WriteFile(fileName, content, 0644)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", fileName, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully created environment file for %s: %s\n", env, fileName)
	},
}

func init() {}
