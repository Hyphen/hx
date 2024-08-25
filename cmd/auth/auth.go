package auth

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/config"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/internal/user"
	"github.com/spf13/cobra"
)

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "auth with hyphen",
	Long:  `Configure sets up environment variables and aliases for the CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {

		if err := login(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}

func login() error {
	oauthService := oauth.DefaultOAuthService()
	token, err := oauthService.StartOAuthServer()
	if err != nil {
		return fmt.Errorf("failed to start OAuth server: %w", err)
	}
	user, error := user.NewService().GetUserInformation()
	if error != nil {
		return fmt.Errorf("failed to get user information: %w", error)
	}

	//Use this as default org
	organizationID := user.Memberships[0].Organization.ID

	if err := config.SaveCredentials(organizationID, token.AccessToken, token.RefreshToken, token.IDToken, token.ExpiryTime); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("Login successful!")
	return nil
}
