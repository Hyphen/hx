package create

import (
	"fmt"

	"github.com/Hyphen/cli/internal/members"
	"github.com/Hyphen/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	firstName string
	lastName  string
	email     string
)

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new member in the organization",
	Long: `Creates a new member in the current organization.

Example:
  hyphen members create --firstName John --lastName Doe --email john.doe@example.com`,
	RunE: runCreate,
}

func init() {
	CreateCmd.Flags().StringVarP(&firstName, "firstName", "f", "", "First name of the new member")
	CreateCmd.Flags().StringVarP(&lastName, "lastName", "l", "", "Last name of the new member")
	CreateCmd.Flags().StringVarP(&email, "email", "e", "", "Email of the new member")

	CreateCmd.MarkFlagRequired("firstName")
	CreateCmd.MarkFlagRequired("lastName")
	CreateCmd.MarkFlagRequired("email")
}

func runCreate(cmd *cobra.Command, args []string) error {
	memberService := members.NewService()
	orgID, err := utils.GetOrganizationID()
	if err != nil {
		return fmt.Errorf("failed to get organization ID: %w", err)
	}

	newMember := members.Member{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	_, err = memberService.CreateMemberForOrg(orgID, newMember)
	if err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}

	cmd.Printf("Member created successfully")

	return nil
}
