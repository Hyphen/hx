package encrypt

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

var envHandler environment.EnviromentHandler

var EncryptCmd = &cobra.Command{
	Use:     "encrypt [FILE]",
	Aliases: []string{"e"},
	Short:   "Encrypt a file",
	Long:    `Encrypts a file containing environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Usage: cloudenv encrypt [FILE]")
			return
		}

		filename := args[0]

		// Initialize the secret key and environment handler if not set
		if envHandler == nil {
			envHandler = environment.Restore()
		}

		// Encrypt environment variables
		encrypted, err := envHandler.EncryptEnvironmentVars(filename)
		if err != nil {
			fmt.Printf("Error encrypting environment variables from file %s: %v\n", filename, err)
			os.Exit(1)
		}

		fmt.Printf("Encrypted data:\n%s\n", encrypted)
	},
}

func setEnvHandler(handler environment.EnviromentHandler) {
	envHandler = handler
}
