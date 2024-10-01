package push

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var PushCmd = &cobra.Command{
	Use:   "push",
	Short: "Upload and encrypt environment .env secrets for a specific environment",
	Long: `
Push and encrypt environment .env secrets for a specific application environment.

This command reads the local .env file corresponding to the specified environment
(e.g., dev, staging, prod), encrypts the variables using the application's secret key,
and uploads them to the specified environment in the Hyphen platform.

Usage:
  hyphen push [flags]

Flags:
  --environment string    Specify the environment to push to (e.g., dev, staging, prod)
  --org string    Specify the organization ID (overrides the default from credentials)

If no environment is specified, it defaults to the "default" environment.
The organization ID is taken from the credentials file if not provided via flag.

Example:
  hyphen push --env production

Note: This command will overwrite existing environment variables in the specified
environment on the Hyphen platform. Make sure you have the necessary permissions
and have reviewed the changes before pushing.
`,
	Run: func(cmd *cobra.Command, args []string) {
		service := newService(env.NewService())

		orgId, err := flags.GetOrganizationID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		prjectId, err := flags.GetProjectID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		appId, err := flags.GetApplicationID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		manifest, err := manifest.Restore()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		envName, err := env.GetEnvironmentID()
		if err != nil {
			cmd.PrintErrf("Error: %s\n", err)
			return
		}

		var envs []string
		if envName != "" && envName != "default" {
			envs = append(envs, envName)
		} else if flags.AllFlag {
			envs, err = service.getLocalEnvsNamesFromFiles()
			if err != nil {
				cprint.Error(cmd, err)
				return
			}
		} else {
			// We're just handling the one special "default" environment
			envs = append(envs, "default")
		}

		if err := service.checkIfLocalEnvsExistAsEnvironments(envs, orgId, prjectId); err != nil {
			cprint.Error(cmd, err)
			return
		}
		for _, envName := range envs {
			e, err := env.GetLocalEnv(envName, manifest)
			if err != nil {
				cprint.Error(cmd, err)
				return
			}
			if err := service.putEnv(orgId, envName, appId, e); err != nil {
				cprint.Error(cmd, err)
				return
			}
		}

		printPushSummary(orgId, *manifest.AppId, envs)
	},
}

type service struct {
	envService env.EnvServicer
}

func newService(envService env.EnvServicer) *service {
	return &service{
		envService,
	}
}

func (s *service) putEnv(orgId, envName, appId string, e env.Env) error {
	if envName == "default" {
		// TODO: This endpoint doesn't yet exist
		fmt.Print("pushing default not yet implemented")
		return nil
	}

	if err := s.envService.PutEnv(orgId, appId, envName, e); err != nil {
		return err
	}
	return nil
}

func (s *service) checkIfLocalEnvsExistAsEnvironments(envs []string, orgId, projectId string) error {
	environments, err := s.envService.ListEnvironments(orgId, projectId, 100, 1)
	if err != nil {
		return err
	}
	mapEnvs := make(map[string]bool)
	for _, env := range environments {
		mapEnvs[env.AlternateID] = true
	}
	for _, env := range envs {
		// skip default, it's not an explicit environment but it's always implicit with .env secrets
		if env == "default" {
			continue
		}

		if _, ok := mapEnvs[env]; !ok {
			return fmt.Errorf("local .env file '.env.%s' does not map to any known project environment", env)
		}
	}

	return nil
}

func (s *service) getLocalEnvsNamesFromFiles() ([]string, error) {
	var envs []string
	envsFiles, err := env.GetEnvsInDirectory()
	if err != nil {
		return []string{}, err
	}
	for _, envFile := range envsFiles {
		envName, err := env.GetEnvNameFromFile(envFile)
		if err != nil {
			return []string{}, err
		}
		envs = append(envs, envName)
	}
	return envs, nil
}

func printPushSummary(orgId, appId string, envs []string) {
	cprint.PrintHeader("--- Push Summary ---")
	cprint.Success("Successfully pushed environment variables")
	cprint.PrintDetail("Organization", orgId)
	cprint.PrintDetail("Application", appId)

	if len(envs) == 1 {
		cprint.PrintDetail("Environment", envs[0])
	} else {
		cprint.PrintDetail("Environments", strings.Join(envs, ", "))
	}

	cprint.PrintDetail("Total environments pushed", fmt.Sprintf("%d", len(envs)))
	fmt.Println()
	cprint.GreenPrint("All environment .env secrets are now securely stored.")
}
