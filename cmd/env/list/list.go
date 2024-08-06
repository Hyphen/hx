package list

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/Hyphen/cli/internal/environment/envvars"
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
	cloudEnvs, err := envHandler.ListCloudEnvironments(pageSize, pageNum)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	localEnvs, err := envHandler.ListLocalEnvironments()
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	if len(cloudEnvs) == 0 && len(localEnvs) == 0 {
		fmt.Println("No environments found.")
		return nil
	}

	t := table.New(os.Stdout)

	t.SetHeaders("Environment ID", "Size", "Variables Count", "Status")

	cloudEnvIds := make(map[string]bool)
	for _, env := range cloudEnvs {
		cloudEnvIds[env.EnvId] = true
	}

	for _, envFile := range localEnvs {
		envId := environment.GetEnvironmentByEnvFile(envFile)
		envData, err := envvars.New(envFile)
		if err != nil {
			return fmt.Errorf("failed to create environment data: %w", err)
		}

		var status string
		if _, exists := cloudEnvIds[envId]; !exists {
			status = "Unsynced"
		} else {
			if envHandler.IsEnvironmentDirty(envId, envData.EnvVarsToArray()) {
				status = "Outdated"
			} else {
				status = "Update"
			}
		}

		t.AddRow(envId, envData.Size, fmt.Sprintf("%d", envData.CountVariables), status)

	}

	localEnvsIds := make(map[string]bool)
	for _, envFile := range localEnvs {
		envId := environment.GetEnvironmentByEnvFile(envFile)
		localEnvsIds[envId] = true
	}

	for _, env := range cloudEnvs {
		var status string
		if _, exists := localEnvsIds[env.EnvId]; !exists {
			status = "On Cloud"
			t.AddRow(env.EnvId, env.Size, fmt.Sprintf("%d", env.CountVariables), status)
		}

	}

	t.Render()

	return nil
}
