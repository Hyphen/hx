package delete

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/internal/members"
	"github.com/Hyphen/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var DeleteCmd = &cobra.Command{
	Use:     "delete <member-id>",
	Aliases: []string{"del"},
	Short:   "Delete a member from the organization",
	Long: `Deletes a member from the current organization.

Example:
  hyphen members delete member123
  hyphen members delete member123 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	memberService := members.NewService()
	orgID, err := utils.GetOrganizationID()
	if err != nil {
		return fmt.Errorf("failed to get organization ID: %w", err)
	}

	memberID := args[0]

	if !utils.YesFlag {
		// Add confirmation prompt if --yes flag is not set
		cmd.Printf("Are you sure you want to delete member %s? (y/N): ", memberID)
		var response string
		_, err = fmt.Scanln(&response)
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}

		// Convert response to lowercase and trim any whitespace
		response = strings.ToLower(strings.TrimSpace(response))

		// Check if the response is affirmative
		if response != "y" && response != "yes" {
			cmd.Println("Operation cancelled.")
			return nil
		}
	}

	err = memberService.DeleteMember(orgID, memberID)
	if err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}

	cmd.Printf("Member %s has been successfully deleted from the organization.\n", memberID)
	return nil
}
