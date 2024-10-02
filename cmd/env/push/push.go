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
	Use:   "push [environment]",
	Short: "Push local environment variables to Hyphen",
	Long: `
The push command uploads local environment variables from .env files to Hyphen.

This command allows you to:
- Push a specific environment by name
- Push all environments found in local .env files
- Encrypt and securely store your environment variables in Hyphen

The command looks for .env files in the current directory with the naming convention .env.[environment_name].

Examples:
  hyphen push production
  hyphen push --all

After pushing, all environment variables will be securely stored in Hyphen and available for use across your project.
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
		}
		return nil
	},
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

		var envs []string
		if len(args) == 1 && args[0] != "default" {
			envs = append(envs, args[0])
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
	if err := s.envService.PutEnvironmentEnv(orgId, appId, envName, e); err != nil {
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
	cprint.Success("Successfully pushed environment variables")
	cprint.Print("")
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
