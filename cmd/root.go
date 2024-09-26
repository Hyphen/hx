package cmd

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/cmd/auth"
	"github.com/Hyphen/cli/cmd/initialize"
	"github.com/Hyphen/cli/cmd/link"
	"github.com/Hyphen/cli/cmd/project"
	"github.com/Hyphen/cli/cmd/pull"
	"github.com/Hyphen/cli/cmd/push"
	"github.com/Hyphen/cli/cmd/setorg"
	"github.com/Hyphen/cli/cmd/update"
	"github.com/Hyphen/cli/cmd/version"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hyphen",
	Short: "cli for hyphen",
	Long:  `hypen is a cli for ...`,
}

func init() {
	rootCmd.AddCommand(version.VersionCmd)
	rootCmd.AddCommand(update.UpdateCmd)
	rootCmd.AddCommand(initialize.InitCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(setorg.SetOrgCmd)
	rootCmd.AddCommand(pull.PullCmd)
	rootCmd.AddCommand(push.PushCmd)
	rootCmd.AddCommand(link.LinkCmd)
	rootCmd.AddCommand(project.ProjectCmd)

	rootCmd.PersistentFlags().StringVar(&flags.OrganizationFlag, "organization", "", "Organization ID (e.g., org_123)")
	rootCmd.PersistentFlags().StringVar(&flags.OrgFlag, "org", "", "Organization ID (e.g., org_123)")
	rootCmd.PersistentFlags().MarkHidden("org") // hide the shorthand to keep help simpler

	rootCmd.PersistentFlags().StringVar(&flags.ProjectFlag, "project", "", "Project ID (e.g., proj_123)")
	rootCmd.PersistentFlags().StringVar(&flags.ProjFlag, "proj", "", "Project ID (e.g., proj_123)")
	rootCmd.PersistentFlags().MarkHidden("proj") // hide the shorthand to keep help simpler

	rootCmd.PersistentFlags().StringVar(&flags.EnvironmentFlag, "env", "", "Project Environment ID (e.g., pevr_12345)")
	rootCmd.PersistentFlags().BoolVarP(&flags.YesFlag, "yes", "y", false, "Automatically answer yes for prompts")
	rootCmd.PersistentFlags().StringVar(&flags.ApiKey, "api-key", "", "API Key")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
