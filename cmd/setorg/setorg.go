package setorg

import (
	"fmt"

	"github.com/Hyphen/cli/cmd/setproject"
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/organizations"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var (
	printer *cprint.CPrinter
)

var SetOrgCmd = &cobra.Command{
	Use:   "set-org <id>",
	Short: "Set the organization ID",
	Long:  `Set the organization ID for the Hyphen CLI to use.`,
	Args:  cobra.MaximumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		organizationID := ""
		if len(args) > 0 {
			organizationID = args[0]
		}
		return SetOrganization(cmd, organizationID)
	},
}

func SetOrganization(cmd *cobra.Command, organizationID string) error {
	printer = cprint.NewCPrinter(flags.VerboseFlag)
	var organization models.Organization
	organizationService := organizations.NewService()
	if organizationID == "" {
		orgs, err := organizationService.ListOrganizations()
		if err != nil {
			return fmt.Errorf("failed to list organizations: %w", err)
		}
		if orgs.Total == 0 {
			return fmt.Errorf("no organizations found")
		}
		if orgs.Total == 1 {
			organization = orgs.Data[0]
			printer.Print(fmt.Sprintf("You only have access to one organization, automatically choosing %s", organization.Name))
		} else {
			// TODO: handle pagination
			choices := make([]prompt.Choice, len(orgs.Data))
			for i, org := range orgs.Data {
				choices[i] = prompt.Choice{
					Id:           org.ID,
					Display:      fmt.Sprintf("%s (%s)", org.Name, org.ID),
					OriginalData: org,
				}
			}
			choice, err := prompt.PromptSelection(choices, "Select an organization:")
			if err != nil {
				return fmt.Errorf("failed to select organization: %w", err)
			}
			organization = choice.OriginalData.(models.Organization)
		}

	} else {
		org, err := organizationService.GetOrganization(organizationID)
		if err != nil {
			return fmt.Errorf("failed to get organization %q: %w", organizationID, err)
		}
		organization = org
	}

	var err error
	err = config.UpsertGlobalOrganizationID(organization.ID)
	if err != nil {
		return fmt.Errorf("failed to update organization ID: %w", err)
	}
	printer.Success(fmt.Sprintf("successfully update default organization to %s (%s)", organization.Name, organization.ID))

	// Make them select a new default project since projects are organization specific
	printer.Print("Because you changed organizations")
	err = setproject.SetProject(cmd, organization.ID, "")
	if err != nil {
		return fmt.Errorf("failed to set default project for new organization: %w", err)
	}

	return nil
}
