package cmd

import (
	"fmt"
	"os"

	// "github.com/Hyphen/cli/cmd/env"
	"github.com/Hyphen/cli/cmd/auth"
	"github.com/Hyphen/cli/cmd/config"
	"github.com/Hyphen/cli/cmd/initialize"
	"github.com/Hyphen/cli/cmd/organization"
	"github.com/Hyphen/cli/cmd/project"
	"github.com/Hyphen/cli/cmd/update"
	"github.com/Hyphen/cli/cmd/version"
	"github.com/Hyphen/cli/pkg/utils"
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
	// rootCmd.AddCommand(env.EnvCmd)
	rootCmd.AddCommand(initialize.InitCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(project.ProjectCmd)
	rootCmd.AddCommand(config.ConfigCmd)
	rootCmd.AddCommand(organization.OrganizationCmd)

	rootCmd.PersistentFlags().StringVar(&utils.OrgFlag, "org", "", "Organization ID (default is used if not provided)")
	rootCmd.PersistentFlags().BoolVarP(&utils.YesFlag, "yes", "y", false, "Automatically answer yes for prompts")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
