package auth

import (
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/spf13/cobra"
)

// AuthCmd represents the auth command
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the server",
	Long: `Authenticate allows users to start a session with the server
by providing credentials.`,
	Run: func(cmd *cobra.Command, args []string) {
		login()
	},
}

func login() {
	oauth.StartOAuthServer()
}
