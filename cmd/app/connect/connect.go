package connect

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/internal/integrations"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

const githubIntegrationType = "github"

var ConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect the app to a GitHub repository",
	Long: `
The connect command links your application to a GitHub repository.

If your organization has connected a GitHub organization, you can link this app
to a new or existing GitHub repository. If GitHub is not yet connected, you will
be provided a link to set up the integration.

Usage:
  hyphen app connect [flags]

Example:
  hyphen app connect
  hyphen app connect --app my-app-id
`,
	RunE: runConnect,
}

func runConnect(cmd *cobra.Command, args []string) error {
	printer := cprint.NewCPrinter(flags.VerboseFlag)

	orgID, err := flags.GetOrganizationID()
	if err != nil {
		return err
	}

	appID, err := flags.GetApplicationID()
	if err != nil {
		return err
	}

	integrationsService := integrations.NewService()

	allIntegrations, err := integrationsService.ListIntegrations(orgID)
	if err != nil {
		return fmt.Errorf("failed to list integrations: %w", err)
	}

	githubIntegrations := filterByType(allIntegrations, githubIntegrationType)

	if len(githubIntegrations) == 0 {
		setupURL := fmt.Sprintf("%s/settings/integrations", apiconf.GetBaseAppUrl())
		printer.Warning("Your organization has not connected a GitHub organization.")
		printer.Print(fmt.Sprintf("To set up the GitHub integration, visit: %s", setupURL))
		return nil
	}

	integration, err := selectIntegration(cmd, githubIntegrations, printer)
	if err != nil {
		return err
	}
	if integration == nil {
		return nil
	}

	return connectToRepository(cmd, integrationsService, orgID, appID, integration, printer)
}

func filterByType(all []models.Integration, intType string) []models.Integration {
	var result []models.Integration
	for _, i := range all {
		if strings.EqualFold(i.Type, intType) {
			result = append(result, i)
		}
	}
	return result
}

func selectIntegration(cmd *cobra.Command, githubIntegrations []models.Integration, printer *cprint.CPrinter) (*models.Integration, error) {
	if len(githubIntegrations) == 1 {
		return &githubIntegrations[0], nil
	}

	choices := make([]prompt.Choice, len(githubIntegrations))
	for i, ig := range githubIntegrations {
		choices[i] = prompt.Choice{
			Id:           ig.ID,
			Display:      ig.Name,
			OriginalData: ig,
		}
	}

	choice, err := prompt.PromptSelection(choices, "Select a GitHub integration:")
	if err != nil {
		return nil, fmt.Errorf("failed to select integration: %w", err)
	}
	if choice.Id == "" {
		printer.Info("Operation cancelled.")
		return nil, nil
	}

	selected := choice.OriginalData.(models.Integration)
	return &selected, nil
}

func connectToRepository(cmd *cobra.Command, svc *integrations.IntegrationsService, orgID, appID string, integration *models.Integration, printer *cprint.CPrinter) error {
	useNew := prompt.PromptYesNo(cmd, "Would you like to connect to a new repository?", false)
	if useNew.IsFlag && !useNew.Confirmed {
		printer.Info("Operation cancelled due to --no flag.")
		return nil
	}

	if useNew.Confirmed {
		return createAndConnect(cmd, svc, orgID, appID, integration, printer)
	}
	return connectExisting(cmd, svc, orgID, appID, integration, printer)
}

func createAndConnect(cmd *cobra.Command, svc *integrations.IntegrationsService, orgID, appID string, integration *models.Integration, printer *cprint.CPrinter) error {
	repoName, err := prompt.PromptString(cmd, "Enter a name for the new repository:")
	if err != nil {
		return err
	}

	printer.Info(fmt.Sprintf("Creating and connecting to new repository '%s'...", repoName))

	connection, err := svc.ConnectAppToRepository(orgID, appID, integration.ID, repoName, true)
	if err != nil {
		return fmt.Errorf("failed to connect app to repository: %w", err)
	}

	printConnectionSummary(connection, printer)
	return nil
}

func connectExisting(cmd *cobra.Command, svc *integrations.IntegrationsService, orgID, appID string, integration *models.Integration, printer *cprint.CPrinter) error {
	repos, err := svc.ListRepositories(orgID, integration.ID)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(repos) == 0 {
		printer.Warning("No repositories found for this integration.")
		return nil
	}

	choices := make([]prompt.Choice, len(repos))
	for i, repo := range repos {
		display := repo.FullName
		if display == "" {
			display = repo.Name
		}
		choices[i] = prompt.Choice{
			Id:           repo.ID,
			Display:      display,
			OriginalData: repo,
		}
	}

	choice, err := prompt.PromptSelection(choices, "Select a repository to connect:")
	if err != nil {
		return fmt.Errorf("failed to select repository: %w", err)
	}
	if choice.Id == "" {
		printer.Info("Operation cancelled.")
		return nil
	}

	repo := choice.OriginalData.(models.Repository)
	printer.Info(fmt.Sprintf("Connecting app to repository '%s'...", repo.FullName))

	connection, err := svc.ConnectAppToRepository(orgID, appID, integration.ID, repo.Name, false)
	if err != nil {
		return fmt.Errorf("failed to connect app to repository: %w", err)
	}

	printConnectionSummary(connection, printer)
	return nil
}

func printConnectionSummary(connection models.AppConnection, printer *cprint.CPrinter) {
	printer.Success("App successfully connected to GitHub repository")
	printer.Print("")
	printer.PrintDetail("Connection ID", connection.ID)
	printer.PrintDetail("Repository", connection.Repository.FullName)
	printer.PrintDetail("Status", connection.Status)
}
