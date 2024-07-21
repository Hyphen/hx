package push

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/Hyphen/cli/internal/environment/envvars"
	"github.com/spf13/cobra"
)

var fileName string

var PushCmd = &cobra.Command{
	Use:     "push [ENVIRONMENT]",
	Aliases: []string{"p"},
	Short:   "Push an existing environmental variable file",
	Long:    `Pushes the contents of an existing environmental variable file to hyphen for the specified environment.`,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		env := "default"

		if len(args) == 1 {
			env = args[0]
		}

		envHandler := environment.Restore()

		if fileName == "" {
			fileName = environment.GetEnvFileByEnvironment(env)
		}

		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			fmt.Printf("File %s does not exist\n", fileName)
			os.Exit(1)
		}

		envVar, err := envvars.New(fileName)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", fileName, err)
			os.Exit(1)
		}

		err = envHandler.UploadEncryptedEnviromentVars(env, envVar)
		if err != nil {
			fmt.Printf("Error uploading environment variables from file %s to environment %s: %v\n", fileName, env, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully pushed contents of %s to hyphen environment %s\n", fileName, env)
	},
}

func init() {
	PushCmd.Flags().StringVarP(&fileName, "file", "f", "", "Specify the file to push. If not set, a file based on the environment will be used.")
}
