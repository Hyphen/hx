package pull

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

// Define variables to hold the values of the env and file flags
var env string
var fileName string

var PullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Decrypt and put environment variables into a file",
	Long: `This command reads the specified environment, decrypts the variables, and 
writes them into the given file.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the default environment file name
		envFile := environment.GetEnvFileByEnvironment(env)

		// If the -f flag is set, use its value as the file name
		if fileName != "" {
			envFile = fileName
		}

		envHandler := environment.Restore()

		// Decrypt environment variables and save them into the specified file
		_, err := envHandler.DecryptedEnviromentVarsIntoAFile(env, envFile)
		if err != nil {
			fmt.Printf("Error saving environment variables to file %s: %v\n", envFile, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully wrote environment variables to file %s\n", envFile)
	},
}

func init() {
	PullCmd.Flags().StringVarP(&env, "environment", "e", "", "Specify the environment")
	PullCmd.Flags().StringVarP(&fileName, "file", "f", "", "Specify the output file name")
}
