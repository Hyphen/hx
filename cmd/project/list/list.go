package list

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/project"
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
	Short:   "List all projects",
	Aliases: []string{"ls"},
	Long: `Lists all projects and their information in a table format.

Examples:
  # List the first page of projects with the default page size
  hyphen project list

  # List the second page of projects with the default page size
  hyphen project list --pageNum 2

  # List projects with a custom page size
  hyphen project list --pageSize 5

  # List projects with a custom page size and page number
  hyphen project list --pageSize 5 --pageNum 2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listProjects(pageSize, pageNum)
	},
}

func init() {
	ListCmd.Flags().IntVarP(&pageNum, "pageNum", "n", 1, "Page number")
	ListCmd.Flags().IntVarP(&pageSize, "pageSize", "s", 10, "Page size")
}

func listProjects(pageSize, pageNum int) error {
	projectService := project.NewService()
	orgID, err := utils.GetOrganizationID()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	projects, err := projectService.GetListProjects(orgID, pageSize, pageNum)
	if err != nil {
		return fmt.Errorf("error fetching projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	t := table.New(os.Stdout)
	t.SetHeaders("Project ID", "Name", "Alternate ID", "Organization")

	for _, p := range projects {
		t.AddRow(p.ID, p.Name, p.AlternateId, p.Organization.Name)
	}

	t.Render()

	return nil
}
