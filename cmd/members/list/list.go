package list

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/members"
	"github.com/Hyphen/cli/pkg/utils"
	"github.com/aquasecurity/table"
	"github.com/spf13/cobra"
)

var (
	pageNum  int
	pageSize int
)

var ListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all members of an organization",
	Aliases: []string{"ls"},
	Long: `Lists all members of an organization in a table format.

Examples:
  # List members of the current organization
  hyphen members list

  # List members with a custom page size
  hyphen members list --pageSize 20

  # List members with a custom page size and page number
  hyphen members list --pageSize 20 --pageNum 2`,
	RunE: runList,
}

func init() {
	ListCmd.Flags().IntVarP(&pageNum, "pageNum", "n", 1, "Page number")
	ListCmd.Flags().IntVarP(&pageSize, "pageSize", "s", 10, "Page size")
}

func runList(cmd *cobra.Command, args []string) error {
	memberService := members.NewService()
	orgID, err := utils.GetOrganizationID()
	if err != nil {
		return fmt.Errorf("failed to get organization ID: %w", err)
	}

	members, err := memberService.ListMembers(orgID)
	if err != nil {
		return fmt.Errorf("failed to list members: %w", err)
	}

	if len(members) == 0 {
		cmd.Println("No members found.")
		return nil
	}

	t := table.New(os.Stdout)
	t.SetHeaders("ID", "First Name", "Last Name", "Email", "Connected Accounts")

	for _, m := range members {
		connectedAccounts := ""
		for i, acc := range m.ConnectedAccounts {
			if i > 0 {
				connectedAccounts += ", "
			}
			connectedAccounts += acc.Type
		}
		t.AddRow(m.ID, m.FirstName, m.LastName, m.Email, connectedAccounts)
	}

	t.Render()
	return nil
}
