package members

import (
	"github.com/Hyphen/cli/cmd/members/create"
	"github.com/Hyphen/cli/cmd/members/delete"
	"github.com/Hyphen/cli/cmd/members/list"
	"github.com/spf13/cobra"
)

var MembersCmd = &cobra.Command{
	Use:   "members",
	Short: "Manage organization members",
	Long:  `Members command allows you to manage and view your organization's members.`,
}

func init() {
	MembersCmd.AddCommand(list.ListCmd)
	MembersCmd.AddCommand(create.CreateCmd)
	MembersCmd.AddCommand(delete.DeleteCmd)
}
