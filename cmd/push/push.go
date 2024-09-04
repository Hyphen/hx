package push

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var PushCmd = &cobra.Command{
	Use:   "push",
	Short: "Upload and encrypt environment variables for a specific environment",
	Long: `
Push and encrypt environment variables for a specific application environment.

This command reads the local .env file corresponding to the specified environment
(e.g., dev, staging, prod), encrypts the variables using the application's secret key,
and uploads them to the specified environment in the Hyphen platform.

Usage:
  hyphen push [flags]

Flags:
  --env string    Specify the environment to push to (e.g., dev, staging, prod)
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

		orgId, err := utils.GetOrganizationID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		envName, err := utils.GetEnvronmentID()
		if err != nil {
			cmd.PrintErrf("Error: %s\n", err)
			return
		}
		var envs []string
		if envName != "" {
			envs = append(envs, envName)
		} else {
			envs, err = service.getLocalEnvsNames()
			if err != nil {
				cprint.Error(cmd, err)
				return
			}
		}

		manifest, err := manifest.Restore()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		err = service.checkForEnvironment(orgId, envName, manifest)
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		e, err := service.getLocalEnv(envName, manifest)
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		if err := service.checkIfLocalEnvsExistAsEnvironments(envs, orgId, manifest); err != nil {
			cprint.Error(cmd, err)
			return
		}

		if err := service.putEnv(orgId, envName, e, manifest); err != nil {
			cprint.Error(cmd, err)
			return
		}

		printPushSummary(orgId, manifest.AppId, envs)

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

func (s *service) checkForEnvironment(orgId, envName string, m manifest.Manifest) error {
	_, exist, err := s.envService.GetEnvironment(orgId, m.AppId, envName)
	if !exist && err == nil {
		return fmt.Errorf("Environment %s not found", envName)
	}
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}

	return nil
}

func (s *service) getLocalEnv(envName string, m manifest.Manifest) (env.Env, error) {
	envFile, err := env.GetFileName(envName)
	if err != nil {
		return env.Env{}, err
	}

	e, err := env.New(envFile)
	if err != nil {
		return env.Env{}, err
	}

	envEncrytedData, err := e.EncryptData(m.GetSecretKey())
	if err != nil {
		return env.Env{}, err
	}
	e.Data = envEncrytedData

	return e, nil
}

func (s *service) putEnv(orgId, envName string, e env.Env, m manifest.Manifest) error {

	if err := s.envService.PutEnv(orgId, m.AppId, envName, e); err != nil {
		return err
	}
	return nil
}

func (s *service) checkIfLocalEnvsExistAsEnvironments(envs []string, orgId string, m manifest.Manifest) error {
	environments, err := s.envService.ListEnvironments(orgId, m.AppId, 100, 1)
	if err != nil {
		return err
	}
	mapEnvs := make(map[string]bool)
	for _, env := range environments {
		mapEnvs[env.Name] = true
	}

	for _, env := range envs {
		if _, ok := mapEnvs[env]; !ok {
			return fmt.Errorf("Environment %s not found in App", env)
		}
	}

	return nil
}

func (s *service) getLocalEnvsNames() ([]string, error) {
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
	cprint.GreenPrint("All environment variables are now securely stored and ready for use.")
}
