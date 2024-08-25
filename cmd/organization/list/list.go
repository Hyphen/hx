package list

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/user"
	"github.com/aquasecurity/table"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all organizations",
	Aliases: []string{"ls"},
	Long:    `Lists all organizations that the user is a member of in a table format.`,
	RunE:    runList,
}

func runList(cmd *cobra.Command, args []string) error {
	userService := user.NewService()
	userInfo, err := userService.GetUserInformation()
	if err != nil {
		return fmt.Errorf("failed to fetch user information: %w", err)
	}

	if len(userInfo.Memberships) == 0 {
		fmt.Println("You are not a member of any organizations.")
		return nil
	}

	t := table.New(os.Stdout)
	t.SetHeaders("Organization ID", "Name")

	for _, membership := range userInfo.Memberships {
		t.AddRow(membership.Organization.ID, membership.Organization.Name)
	}

	t.Render()
	return nil
}
