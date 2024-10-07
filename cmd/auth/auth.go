package auth

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Hyphen",
	Args:  cobra.NoArgs,
	Long:  `Authenticate and set up credentials for the Hyphen CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := login(); err != nil {
			cprint.Error(cmd, err)
			return
		}
	},
}

func init() {
	AuthCmd.PersistentFlags().StringVar(&flags.SetApiKeyFlag, "set-api-key", "", "Authenticate using API key provided inline")
	AuthCmd.PersistentFlags().BoolVar(&flags.UseApiKeyFlag, "use-api-key", false, "Authenticate using an API key provided via prompt or HYPHEN_API_KEY env variable")
}

func login() error {
	if !flags.UseApiKeyFlag && flags.SetApiKeyFlag == "" {
		oauthService := oauth.DefaultOAuthService()
		token, err := oauthService.StartOAuthServer()
		if err != nil {
			return fmt.Errorf("failed to start OAuth server: %w", err)
		}

		if flags.VerboseFlag {
			cprint.Success("OAuth server started successfully")
		}

		m := manifest.Manifest{
			ManifestConfig: manifest.ManifestConfig{
				HyphenAccessToken:  &token.AccessToken,
				HyphenRefreshToken: &token.RefreshToken,
				HypenIDToken:       &token.IDToken,
				ExpiryTime:         &token.ExpiryTime,
			},
		}

		if err := manifest.UpsertGlobalManifest(m); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		if flags.VerboseFlag {
			cprint.Success("Credentials saved successfully")
		}
	} else {
		if flags.UseApiKeyFlag {
			if flags.SetApiKeyFlag != "" {
				return fmt.Errorf("cannot use both --use-api-key and --set-api-key flags together")
			}
		}

		var apiKey string
		if flags.UseApiKeyFlag {
			// Check against the env first
			if os.Getenv("HYPHEN_API_KEY") != "" {
				apiKey = os.Getenv("HYPHEN_API_KEY")
			} else {
				// password prompt
				var err error
				apiKey, err = prompt.PromptPassword("Paste in your API key and hit enter: ")
				if err != nil {
					return fmt.Errorf("failed to read API key: %w", err)
				}
			}
		}

		if flags.SetApiKeyFlag != "" {
			apiKey = flags.SetApiKeyFlag
		}

		m := manifest.Manifest{
			ManifestConfig: manifest.ManifestConfig{
				HyphenAccessToken: &apiKey,
			},
		}

		if err := manifest.UpsertGlobalManifest(m); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		if flags.VerboseFlag {
			cprint.Success("Credentials saved successfully")
		}
	}

	user, err := user.NewService().GetUserInformation()
	if err != nil {
		return fmt.Errorf("failed to get user information: %w", err)
	}

	organizationID := user.Memberships[0].Organization.ID

	projectService := projects.NewService(organizationID)
	projectList, err := projectService.ListProjects()
	if err != nil {
		return err
	}
	if len(projectList) == 0 {
		return fmt.Errorf("no projects found")
	}

	defaultProject := projectList[0]

	mc := manifest.ManifestConfig{
		ProjectId:          defaultProject.ID,
		ProjectName:        &defaultProject.Name,
		ProjectAlternateId: &defaultProject.AlternateID,
		OrganizationId:     organizationID,
		ExpiryTime:         &token.ExpiryTime,
		HyphenAccessToken:  &token.AccessToken,
		HyphenRefreshToken: &token.RefreshToken,
		HypenIDToken:       &token.IDToken,
		AppId:              nil,
		AppName:            nil,
		AppAlternateId:     nil,
	}

	if _, err := manifest.GlobalInitialize(mc); err != nil {
		return err
	}

	printAuthenticationSummary(&user, organizationID, *defaultProject.ID)
	return nil
}

func printAuthenticationSummary(user *user.UserInfo, organizationID string, projectID string) {
	if flags.VerboseFlag {
		cprint.PrintHeader("Authentication Summary")
		cprint.Success("Login successful!")
		cprint.Print("") // Add an empty line for better spacing
		cprint.PrintDetail("User", user.Memberships[0].Email)
		cprint.PrintDetail("Organization ID", organizationID)
		cprint.PrintDetail("Default Project ID", projectID)
		cprint.Print("") // Add an empty line for better spacing
	}
	cprint.GreenPrint("You are now authenticated and ready to use Hyphen CLI.")
}
