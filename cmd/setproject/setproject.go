package setproject

import (
	"fmt"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/helpers"
	"github.com/spf13/cobra"
)

// printer is a package-level var because flags.VerboseFlag isn't bound
// until cobra parses flags at runtime, so it can't be initialized in
// init(). It's lazily initialized at each entry point —
// SetProjectCmd.RunE and SetProject — with a nil-check on the latter
// so it doesn't clobber state set on the printer earlier in the
// command's lifecycle.
// TODO: thread a *CPrinter through the call signatures and drop the
// package-level mutable state.
var (
	printer *cprint.CPrinter
)

var SetProjectCmd = &cobra.Command{
	Use:   "set-project <id/alternate_id>",
	Short: "Set the default project",
	Long:  `Set the default project for the Hyphen CLI to use.`,
	Args:  cobra.MaximumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		organizationID, err := flags.GetOrganizationID()
		projectID := ""
		if len(args) > 0 {
			projectID = args[0]
		}
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to get organization ID: %v", err))
			return err
		}

		return SetProject(cmd, organizationID, projectID)

	},
}

func SetProject(cmd *cobra.Command, organizationID, projectID string) error {
	// SetProject is called both from SetProjectCmd.RunE (which already
	// initializes printer) and directly from setorg.SetOrganization
	// (where RunE never fires). Initialize only if not already set so
	// we don't clobber state.
	if printer == nil {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
	}
	projectService := projects.NewService(organizationID)
	var project models.Project
	if projectID == "" {
		proj, err := helpers.SelectProject(organizationID, "Select a default project:")
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to select project: %v", err))
			return err
		}
		project = proj
	} else {
		proj, err := projectService.GetProject(projectID)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to get project %q: %v", projectID, err))
			return err
		}
		project = proj
	}

	if project.ID == nil {
		err := fmt.Errorf("project %q has no ID; cannot set as default", project.Name)
		printer.Error(cmd, err)
		return err
	}

	err := config.UpsertGlobalProject(project)
	if err != nil {
		printer.Error(cmd, fmt.Errorf("failed to update default project: %v", err))
		return err
	}
	printer.Success(fmt.Sprintf("successfully updated default project to %s (%s)", project.Name, *project.ID))

	return nil
}
