package pull

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

var fileName string

var PullCmd = &cobra.Command{
	Use:   "pull [environment]",
	Short: "Decrypt and put environment variables into a file",
	Long: `This command reads the specified environment, decrypts the variables, and 
writes them into the given file.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		env := "default"

		if len(args) == 1 {
			env = args[0]
		}

		envFile := environment.GetEnvFileByEnvironment(env)

		if fileName != "" {
			envFile = fileName
		}

		envHandler := environment.Restore()

		_, err := envHandler.DecryptedEnvironmentVarsIntoAFile(env, envFile)
		if err != nil {
			fmt.Printf("Error saving environment variables to file %s: %v\n", envFile, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully wrote environment variables to file %s\n", envFile)
	},
}

func init() {
	PullCmd.Flags().StringVarP(&fileName, "file", "f", "", "Specify the output file name")
}
