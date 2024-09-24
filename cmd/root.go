package cmd

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/cmd/auth"
	"github.com/Hyphen/cli/cmd/initialize"
	"github.com/Hyphen/cli/cmd/pull"
	"github.com/Hyphen/cli/cmd/push"
	"github.com/Hyphen/cli/cmd/update"
	"github.com/Hyphen/cli/cmd/useorg"
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
	rootCmd.AddCommand(initialize.InitCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(useorg.UseOrgCmd)
	rootCmd.AddCommand(pull.PullCmd)
	rootCmd.AddCommand(push.PushCmd)

	rootCmd.PersistentFlags().StringVar(&utils.OrgFlag, "org", "", "Organization ID (e.g., org_123)")
	rootCmd.PersistentFlags().StringVar(&utils.ProjFlag, "proj", "", "Project ID (e.g., proj_123)")
	rootCmd.PersistentFlags().StringVar(&utils.EnvironmentFlag, "env", "", "Environment ID (e.g., env_12345)")
	rootCmd.PersistentFlags().BoolVarP(&utils.YesFlag, "yes", "y", false, "Automatically answer yes for prompts")
	rootCmd.PersistentFlags().StringVar(&utils.ApiKey, "app-key", "", "API Key")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
