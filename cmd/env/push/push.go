package push

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	// Assuming `environment` package provides necessary methods
	"github.com/Hyphen/cli/internal/environment"
)

// Define a variable to hold the value of the file flag
var fileName string

var PushCmd = &cobra.Command{
	Use:     "push [ENVIRONMENT]",
	Aliases: []string{"p"},
	Short:   "Push an existing environmental variable file",
	Long:    `Pushes the contents of an existing environmental variable file to heyphen for the specified environment.`,
	Args:    cobra.ExactArgs(1), // Ensure exactly one argument is provided, which is the environment
	Run: func(cmd *cobra.Command, args []string) {
		// Get the environment from the first command-line argument
		env := args[0]
		envHandler := environment.Restore()

		// If the -f flag is set, use its value as the file name
		if fileName == "" {
			// If -f flag is not set, get the default environment file name
			fileName = environment.GetEnvFileByEnvironment(env)
		}

		// Check if the file exists
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			fmt.Printf("File %s does not exist\n", fileName)
			os.Exit(1)
		}

		// Read the file contents
		content, err := os.ReadFile(fileName)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", fileName, err)
			os.Exit(1)
		}

		// Upload the encrypted environment variables
		err = envHandler.UploadEncryptedEnviromentVars(env, string(content))
		if err != nil {
			fmt.Printf("Error uploading environment variables from file %s to environment %s: %v\n", fileName, env, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully pushed contents of %s to heyphen environment %s\n", fileName, env)
	},
}

func init() {
	// Register the file flag
	PushCmd.Flags().StringVarP(&fileName, "file", "f", "", "Specify the file to push. If not set, a file based on the environment will be used.")
}
