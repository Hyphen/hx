package setorg

import (
	"fmt"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	globalFlag bool
	printer    *cprint.CPrinter
)

var SetOrgCmd = &cobra.Command{
	Use:   "set-org <id>",
	Short: "Set the organization ID",
	Long:  `Set the organization ID for the Hyphen CLI to use.`,
	Args:  cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		orgID := args[0]
		var err error

		projectService := projects.NewService(orgID)
		projectList, err := projectService.ListProjects()
		if err != nil {
			return err
		}
		if len(projectList) == 0 {
			return fmt.Errorf("no projects found")
		}
		defaultProject := projectList[0]

		if globalFlag {
			err = config.UpsertGlobalOrganizationID(orgID)
		} else {
			err = config.UpsertOrganizationID(orgID)
		}
		if err != nil {
			return fmt.Errorf("failed to update organization ID: %w", err)
		}

		if globalFlag {
			err = config.UpsertGlobalProject(defaultProject)
		} else {
			err = config.UpsertProject(defaultProject)
		}
		if err != nil {
			return fmt.Errorf("failed to update project ID: %w", err)
		}

		printOrgUpdateSuccess(orgID, globalFlag)
		return nil
	},
}

func init() {
	SetOrgCmd.Flags().BoolVar(&globalFlag, "global", false, "Set the organization ID globally")
}

func printOrgUpdateSuccess(orgID string, isGlobal bool) {
	printer.PrintHeader("--- Organization Update ---")
	if isGlobal {
		printer.Success("Successfully updated global organization ID")
	} else {
		printer.Success("Successfully updated organization ID")
	}
	printer.PrintDetail("New Organization ID", orgID)
	fmt.Println()
	if isGlobal {
		printer.GreenPrint("Hyphen CLI is now set to use the new organization globally.")
	} else {
		printer.GreenPrint("Hyphen CLI is now set to use the new organization.")
	}
}
