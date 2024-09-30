package auth

import (
	"fmt"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
)

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Hyphen",
	Long:  `Authenticate and set up credentials for the Hyphen CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := login(); err != nil {
			cprint.Error(cmd, err)
			return
		}
	},
}

func login() error {
	cprint.PrintHeader("Hyphen Authentication Process")

	oauthService := oauth.DefaultOAuthService()
	token, err := oauthService.StartOAuthServer()
	if err != nil {
		return fmt.Errorf("failed to start OAuth server: %w", err)
	}

	cprint.Success("OAuth server started successfully")

	ms := manifest.ManifestSecret{
		HyphenAccessToken:  token.AccessToken,
		HyphenRefreshToken: token.RefreshToken,
		HypenIDToken:       token.IDToken,
		ExpiryTime:         token.ExpiryTime,
	}
	if err := manifest.UpsertGlobalSecret(ms); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	cprint.Success("Credentials saved successfully")

	user, err := user.NewService().GetUserInformation()
	if err != nil {
		return fmt.Errorf("failed to get user information: %w", err)
	}

	var organizationID string
	organizationID = user.Memberships[0].Organization.ID

	projectService := projects.NewService(organizationID)
	projectList, err := projectService.ListProjects()
	if err != nil {
		return err
	}
	if len(projectList) == 0 {
		return fmt.Errorf("No projects found")
	}

	defaultProject := projectList[0]

	mc := manifest.ManifestConfig{
		ProjectId:          defaultProject.ID,
		ProjectName:        &defaultProject.Name,
		ProjectAlternateId: &defaultProject.AlternateID,
		OrganizationId:     organizationID,
		AppId:              nil,
		AppName:            nil,
		AppAlternateId:     nil,
	}

	if _, err := manifest.GlobalInitialize(mc, ms); err != nil {
		return err
	}

	printAuthenticationSummary(&user, organizationID)
	return nil
}

func printAuthenticationSummary(user *user.UserInfo, organizationID string) {
	cprint.PrintHeader("Authentication Summary")
	cprint.Success("Login successful!")
	cprint.Print("") // Add an empty line for better spacing
	cprint.PrintDetail("User", user.Memberships[0].Email)
	cprint.PrintDetail("Organization ID", organizationID)
	cprint.Print("") // Add an empty line for better spacing
	cprint.GreenPrint("You are now authenticated and ready to use Hyphen CLI.")
}
