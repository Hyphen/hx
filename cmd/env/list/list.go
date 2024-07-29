package list

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/aquasecurity/table"
	"github.com/spf13/cobra"
)

var pageSize int
var pageNum int

var ListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all environments",
	Aliases: []string{"ls"},
	Long: `Lists all environments and their information in a table format.

Examples:
  # List the first page of environments with the default page size
  hyphen env list

  # List the second page of environments with the default page size
  hyphen env list --pageNum 2

  # List environments with a custom page size
  hyphen env list --pageSize 5

  # List environments with a custom page size and page number
  hyphen env list --pageSize 5 --pageNum 2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listEnvironments(pageSize, pageNum)
	},
}

func init() {
	ListCmd.Flags().IntVarP(&pageSize, "pageSize", "s", 10, "Number of environments per page")
	ListCmd.Flags().IntVarP(&pageNum, "pageNum", "n", 1, "Page number to display")
}

func listEnvironments(pageSize, pageNum int) error {
	envHandler := environment.Restore()
	envs, err := envHandler.ListEnvironments(pageSize, pageNum)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	if len(envs) == 0 {
		fmt.Println("No environments found.")
		return nil
	}

	t := table.New(os.Stdout)

	t.SetHeaders("Environment ID", "Size", "Variables Count")

	for _, env := range envs {
		t.AddRow(env.EnvId, env.Size, fmt.Sprintf("%d", env.CountVariables))
	}

	t.Render()

	return nil
}
