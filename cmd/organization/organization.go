package organization

import (
	"github.com/Hyphen/cli/cmd/organization/list"
	"github.com/spf13/cobra"
)

var OrganizationCmd = &cobra.Command{
	Use:     "organization",
	Aliases: []string{"org"},
	Short:   "Manage organizations",
	Long:    `Organization command allows you to manage and view your Hyphen organizations.`,
}

func init() {
	OrganizationCmd.AddCommand(list.ListCmd)
}
