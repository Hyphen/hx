package initialize

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/config"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/spf13/cobra"
)

// ConfigureCmd represents the configure command
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "init hyphen cli",
	Long:  `Configure sets up environment variables and aliases for the CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {

		if err := addAlias("envx", "hyphen env"); err != nil {
			fmt.Println("Warning: ", err)
		} else {
			fmt.Println("Please source the console or close and open the terminal to use the alias 'envx'")
		}

		if err := login(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}

func login() error {
	token, err := oauth.StartOAuthServer()
	if err != nil {
		return err
	}
	if err := config.SaveCredentials(token.AccessToken, token.RefreshToken, token.IDToken, token.ExpiryTime); err != nil {
		return err
	}

	return nil
}
