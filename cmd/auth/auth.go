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

var printer *cprint.CPrinter

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Hyphen",
	Args:  cobra.NoArgs,
	Long: `Authenticate and set up credentials for the Hyphen CLI.

This command allows you to log in to your Hyphen account via OAuth or an API key, and securely store your credentials for future CLI interactions.

The authentication process supports two methods:
- OAuth Login (default): This method will open a browser window and prompt you to log in using your Hyphen credentials.
- API Key Login: If you prefer or are required to use an API key, you can authenticate by providing the key either via an environment variable, an inline flag, or interactively via a prompt.

Examples:
	hyhen auth
	hyphen auth --use-api-key # This will read check for HYPHEN_API_KEY in the environment and prompt if not found
	hyphen auth --set-api-key YOURKEY1234
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := login(cmd); err != nil {
			printer.Error(cmd, err)
			return
		}
	},
}

func init() {
	AuthCmd.PersistentFlags().StringVar(&flags.SetApiKeyFlag, "set-api-key", "", "Authenticate using API key provided inline")
	AuthCmd.PersistentFlags().BoolVar(&flags.UseApiKeyFlag, "use-api-key", false, "Authenticate using an API key provided via prompt or HYPHEN_API_KEY env variable")
	printer = cprint.NewCPrinter(flags.VerboseFlag)
}

func login(cmd *cobra.Command) error {

	var accessToken *string
	var refreshToken *string
	var idToken *string
	var expiryTime *int64
	var apiKey *string

	// Check for standard login flow (oauth)
	if !flags.UseApiKeyFlag && flags.SetApiKeyFlag == "" {
		oauthService := oauth.DefaultOAuthService()
		token, err := oauthService.StartOAuthServer()
		if err != nil {
			return fmt.Errorf("failed to start OAuth server: %w", err)
		}

		if flags.VerboseFlag {
			printer.Success("OAuth server started successfully")
		}

		accessToken = &token.AccessToken
		refreshToken = &token.RefreshToken
		idToken = &token.IDToken
		expiryTime = &token.ExpiryTime

		mc := manifest.Config{
			HyphenAccessToken:  accessToken,
			HyphenRefreshToken: refreshToken,
			HypenIDToken:       idToken,
			ExpiryTime:         expiryTime,
		}

		if err := manifest.UpsertGlobalConfig(mc); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		if flags.VerboseFlag {
			printer.Success("Credentials saved successfully")
		}
	} else { // API key login flow
		if flags.UseApiKeyFlag {
			if flags.SetApiKeyFlag != "" {
				return fmt.Errorf("cannot use both --use-api-key and --set-api-key flags together")
			}
		}

		if flags.UseApiKeyFlag {
			// Check against the env first
			if os.Getenv("HYPHEN_API_KEY") != "" {
				key := os.Getenv("HYPHEN_API_KEY")
				apiKey = &key
			} else {
				// password prompt
				var err error
				key, err := prompt.PromptPassword(cmd, "Paste in your API key and hit enter: ")
				if err != nil {
					return fmt.Errorf("failed to read API key: %w", err)
				}
				apiKey = &key
			}
		} else if flags.SetApiKeyFlag != "" {
			apiKey = &flags.SetApiKeyFlag
		}

		mc := manifest.Config{
			HyphenAPIKey: apiKey,
		}

		if err := manifest.UpsertGlobalConfig(mc); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		if flags.VerboseFlag {
			printer.Success("Credentials saved successfully")
		}
	}

	executionContext, err := user.NewService().GetExecutionContext()
	if err != nil {
		return fmt.Errorf("failed to get user information: %w", err)
	}

	organizationID := executionContext.Member.Organization.ID

	projectService := projects.NewService(organizationID)
	projectList, err := projectService.ListProjects()
	if err != nil {
		return err
	}
	if len(projectList) == 0 {
		return fmt.Errorf("no projects found")
	}

	defaultProject := projectList[0]

	mc := manifest.Config{
		ProjectId:          defaultProject.ID,
		ProjectName:        &defaultProject.Name,
		ProjectAlternateId: &defaultProject.AlternateID,
		OrganizationId:     organizationID,
		ExpiryTime:         expiryTime,
		HyphenAccessToken:  accessToken,
		HyphenRefreshToken: refreshToken,
		HypenIDToken:       idToken,
		HyphenAPIKey:       apiKey,
		AppId:              nil,
		AppName:            nil,
		AppAlternateId:     nil,
	}

	if err := manifest.GlobalInitializeConfig(mc); err != nil {
		return err
	}

	printAuthenticationSummary(&executionContext, organizationID, *defaultProject.ID)
	return nil
}

func printAuthenticationSummary(user *user.ExecutionContext, organizationID string, projectID string) {
	if flags.VerboseFlag {
		printer.PrintHeader("Authentication Summary")
		printer.Success("Login successful!")
		printer.Print("") // Add an empty line for better spacing
		printer.PrintDetail("User", user.User.Name)
		printer.PrintDetail("Organization ID", organizationID)
		printer.PrintDetail("Default Project ID", projectID)
		printer.Print("") // Add an empty line for better spacing
	}
	printer.GreenPrint("You are now authenticated and ready to use Hyphen CLI.")
}
