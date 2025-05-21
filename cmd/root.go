package cmd

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/cmd/app"
	"github.com/Hyphen/cli/cmd/auth"
	"github.com/Hyphen/cli/cmd/build"
	"github.com/Hyphen/cli/cmd/deploy"
	"github.com/Hyphen/cli/cmd/env"
	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/Hyphen/cli/cmd/initialize"
	"github.com/Hyphen/cli/cmd/initproject"
	"github.com/Hyphen/cli/cmd/link"
	"github.com/Hyphen/cli/cmd/project"
	"github.com/Hyphen/cli/cmd/setorg"
	"github.com/Hyphen/cli/cmd/setproject"
	"github.com/Hyphen/cli/cmd/update"
	"github.com/Hyphen/cli/cmd/version"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hyphen",
	Short: "cli for Hyphen",
	Long: `
Hyphen CLI

The Hyphen CLI is a command-line interface for managing your Hyphen projects, environments, applications, and more. It provides a set of commands to interact with various resources in your Hyphen account.`,
}

func init() {
	rootCmd.AddCommand(version.VersionCmd)
	rootCmd.AddCommand(update.UpdateCmd)
	rootCmd.AddCommand(initialize.InitCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(setorg.SetOrgCmd)
	rootCmd.AddCommand(setproject.SetProjectCmd)
	rootCmd.AddCommand(pull.PullCmd)
	rootCmd.AddCommand(push.PushCmd)
	rootCmd.AddCommand(link.LinkCmd)
	rootCmd.AddCommand(app.AppCmd)
	rootCmd.AddCommand(project.ProjectCmd)
	rootCmd.AddCommand(env.EnvCmd)
	rootCmd.AddCommand(initproject.InitProjectCmd)
	rootCmd.AddCommand(deploy.DeployCmd)
	rootCmd.AddCommand(build.BuildCmd)

	// Override the default completion command with a hidden no-op command
	rootCmd.AddCommand(&cobra.Command{
		Use:    "completion",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			// No-op
		},
	})

	rootCmd.PersistentFlags().StringVarP(&flags.OrganizationFlag, "organization", "o", "", "Organization ID (e.g., org_123)")
	rootCmd.PersistentFlags().StringVarP(&flags.ProjectFlag, "project", "p", "", "Project ID (e.g., proj_123)")

	rootCmd.PersistentFlags().BoolVarP(&flags.YesFlag, "yes", "y", false, "Automatically answer yes for prompts")
	rootCmd.PersistentFlags().BoolVarP(&flags.NoFlag, "no", "n", false, "Automatically answer no for prompts")
	rootCmd.PersistentFlags().BoolVarP(&flags.VerboseFlag, "verbose", "v", false, "Enable more verbose output")

	// Hidden --dev flag for interacting against the Hyphen development environment
	rootCmd.PersistentFlags().BoolVar(&flags.DevFlag, "dev", false, "Use the Hyphen development environment")
	rootCmd.PersistentFlags().MarkHidden("dev")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
