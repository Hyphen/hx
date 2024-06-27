package create

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

// Define a variable to hold the value of the file flag
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
	Args: cobra.ExactArgs(1), // Ensure exactly one argument is provided, which is the environment
	Run: func(cmd *cobra.Command, args []string) {
		// Get the environment from the first command-line argument
		env := args[0]

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
