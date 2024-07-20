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

The command will create a file named based on the environment, such as '.env.dev' or '.env.prod'.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		env := ""

		if len(args) == 1 {
			env = args[0]
		}
		env, err := environment.GetEnvName(env)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fileName = environment.GetEnvFileByEnvironment(env)

		// Check if the file already exists to avoid overwriting
		if _, err := os.Stat(fileName); err == nil {
			fmt.Printf("File %s already exists. Use a different environment or delete the existing file.\n", fileName)
			os.Exit(1)
		}

		content := []byte("# Environment variables\nKEY=Value\n")

		err = os.WriteFile(fileName, content, 0644)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", fileName, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully created environment file for %s: %s\n", env, fileName)
	},
}

func init() {}
