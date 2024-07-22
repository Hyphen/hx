package encrypt

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

var envHandler environment.EnvironmentHandler

var EncryptCmd = &cobra.Command{
	Use:     "encrypt [FILE]",
	Aliases: []string{"e"},
	Short:   "Encrypt a file",
	Long: `Encrypts a file containing environment variables.

Examples:
  # Encrypt a default .env file
  hyrule env encrypt .env

  # Encrypt a specific environment file
  hyrule env encrypt .env.production

  # Encrypt a custom named environment file
  hyrule env encrypt my-custom-env-file.txt

Note: The encrypted output will be displayed in the console. Make sure to handle this sensitive information securely.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Usage: hyphen env encrypt [FILE]")
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

func setEnvHandler(handler environment.EnvironmentHandler) {
	envHandler = handler
}
