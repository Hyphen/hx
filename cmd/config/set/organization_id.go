package set

import (
	"fmt"

	"github.com/Hyphen/cli/config"
	"github.com/spf13/cobra"
)

var organizationIDCmd = &cobra.Command{
	Use:   "organization-id <id>",
	Short: "Set the organization ID",
	Long:  `Set the organization ID for the Hyphen CLI.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgID := args[0]
		err := config.UpdateOrganizationID(orgID)
		if err != nil {
			return fmt.Errorf("failed to update organization ID: %w", err)
		}
		fmt.Printf("Organization ID updated to: %s\n", orgID)
		return nil
	},
}
