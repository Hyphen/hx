package list

//
// import (
// 	"fmt"
// 	"os"
//
// 	"github.com/Hyphen/cli/internal/environment"
// 	"github.com/aquasecurity/table"
// 	"github.com/spf13/cobra"
// )
//
// var ListCmd = &cobra.Command{
// 	Use:     "list",
// 	Short:   "List all environments",
// 	Aliases: []string{"ls"},
// 	Long:    `This command lists all environments and their information in a table format.`,
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		return listEnvironments()
// 	},
// }
//
// func listEnvironments() error {
// 	// Create a new Environment instance
// 	e := environment.Restore()
// 	// Call ListEnvironments method
// 	envs, err := e.ListEnvironments()
// 	if err != nil {
// 		return fmt.Errorf("failed to list environments: %w", err)
// 	}
//
// 	// Create a new table
// 	t := table.New(os.Stdout)
//
// 	// Set headers
// 	t.SetHeaders("Environment ID", "Size", "Variables Count", "Data")
//
// 	// Add rows
// 	for _, env := range envs {
// 		t.AddRow(env.EnvId, env.Size, fmt.Sprintf("%d", env.CountVariables), env.Data)
// 	}
//
// 	// Render the table
// 	t.Render()
//
// 	return nil
// }
