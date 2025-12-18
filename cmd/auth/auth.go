package auth

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/cmd/setorg"
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/helpers"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/Hyphen/cli/pkg/toggle"
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
	hyphen auth
	hyphen auth --use-api-key # This will read check for HYPHEN_API_KEY in the environment and prompt if not found
	hyphen auth --set-api-key YOURKEY1234
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		if err := login(cmd); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	AuthCmd.PersistentFlags().StringVar(&flags.SetApiKeyFlag, "set-api-key", "", "Authenticate using API key provided inline")
	AuthCmd.PersistentFlags().BoolVar(&flags.UseApiKeyFlag, "use-api-key", false, "Authenticate using an API key provided via prompt or HYPHEN_API_KEY env variable")
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

		mc := config.Config{
			HyphenAccessToken:  accessToken,
			HyphenRefreshToken: refreshToken,
			HypenIDToken:       idToken,
			ExpiryTime:         expiryTime,
		}

		if err := config.UpsertGlobalConfig(mc); err != nil {
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

		mc := config.Config{
			HyphenAPIKey: apiKey,
		}

		if err := config.UpsertGlobalConfig(mc); err != nil {
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

	if !helpers.IsInTerminal() {
		organizationID := executionContext.Member.Organization.ID
		mc := config.Config{
			OrganizationId: organizationID,
		}

		projectService := projects.NewService(organizationID)
		projects, _ := projectService.ListProjects()
		if projects != nil || len(projects) > 0 {
			proj := projects[0]
			mc.ProjectId = proj.ID
			mc.ProjectName = &proj.Name
			mc.ProjectAlternateId = &proj.AlternateID
		}
		if err := config.GlobalInitializeConfig(mc); err != nil {
			return err
		}

	} else {
		err = setorg.SetOrganization(cmd, "")
		if err != nil {
			return fmt.Errorf("failed to set organization: %w", err)
		}
	}

	toggle.HandleAuth(executionContext)

	return nil
}
