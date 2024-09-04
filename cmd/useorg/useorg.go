package useorg

import (
	"fmt"

	"github.com/Hyphen/cli/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var UseOrgCmd = &cobra.Command{
	Use:   "use-org <id>",
	Short: "Set the organization ID",
	Long:  `Set the organization ID for the Hyphen CLI.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgID := args[0]
		err := config.UpdateOrganizationID(orgID)
		if err != nil {
			return fmt.Errorf("failed to update organization ID: %w", err)
		}
		printOrgUpdateSuccess(orgID)
		return nil
	},
}

// Color definitions
var (
	green = color.New(color.FgGreen, color.Bold).SprintFunc()
	cyan  = color.New(color.FgCyan).SprintFunc()
	white = color.New(color.FgWhite, color.Bold).SprintFunc()
)

func printOrgUpdateSuccess(orgID string) {
	fmt.Println("\n--- Organization Update ---")
	fmt.Printf("%s %s\n", green("✅"), white("Successfully updated organization ID"))
	fmt.Printf("   %s %s\n", white("New Organization ID:"), cyan(orgID))
	fmt.Println("\n" + green("Hyphen CLI is now set to use the new organization."))
}
