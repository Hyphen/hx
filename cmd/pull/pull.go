package pull

import (
	"fmt"

	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var PullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Retrieve and decrypt environment variables for a specific environment",
	Long: `
Pull and decrypt environment variables for a specific application environment.

This command retrieves the encrypted environment variables from the specified
environment (e.g., dev, staging, prod) and decrypts them using the application's
secret key. The decrypted variables are then saved to a local .env file corresponding
to the environment name.

Usage:
  hyphen pull [flags]

Flags:
  --env string    Specify the environment to pull from (e.g., dev, staging, prod)
  --org string    Specify the organization ID (overrides the default from credentials)

If no environment is specified, it defaults to the "default" environment.
The organization ID is taken from the credentials file if not provided via flag.

Example:
  hyphen pull --env staging
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

		manifest, err := manifest.Restore()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		envName, err := env.GetEnvronmentID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		if envName == "" {
			pulledEnvs, err := service.getAllEnvsAndDecryptIntoFiles(orgId, prjectId, manifest.GetSecretKey())
			if err != nil {
				cprint.Error(cmd, err)
				return
			}

			printPullSummary(prjectId, pulledEnvs)
		} else {
			err = service.checkForEnvironment(orgId, envName, prjectId)
			if err != nil {
				cprint.Error(cmd, err)
				return
			}
			if err = service.saveDecryptedEnvIntoFile(orgId, envName, prjectId, manifest.GetSecretKey()); err != nil {
				cprint.Error(cmd, err)
				return
			}

			printPullSummary(prjectId, []string{envName})
		}

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

func (s *service) checkForEnvironment(orgId, envName, projectId string) error {
	_, exist, err := s.envService.GetEnvironment(orgId, projectId, envName)
	if !exist && err == nil {
		return fmt.Errorf("Environment %s not found", envName)
	}
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}

	return nil
}

func (s *service) saveDecryptedEnvIntoFile(orgId, envName, projectId string, secretKey *secretkey.SecretKey) error {
	e, err := s.envService.GetEnv(orgId, projectId, envName)
	if err != nil {
		return err
	}

	envFile, err := env.GetFileName(envName)
	if err != nil {
		return err
	}

	if _, err = e.DecryptVarsAndSaveIntoFile(envFile, secretKey); err != nil {
		return fmt.Errorf("Failed to save decrypted environment: %s, variables to file: %s", envName, envFile)
	}

	return nil
}

func (s *service) getAllEnvsAndDecryptIntoFiles(orgId, projectId string, secretkey *secretkey.SecretKey) ([]string, error) {
	allEnvs, err := s.envService.ListEnvs(orgId, projectId, 100, 1)
	if err != nil {
		return nil, err
	}
	var pulledEnvs []string
	for _, e := range allEnvs {
		envName := e.ProjectEnv.AlternateID
		if err := s.saveDecryptedEnvIntoFile(orgId, envName, projectId, secretkey); err != nil {
			return pulledEnvs, err
		}
		pulledEnvs = append(pulledEnvs, envName)
	}
	return pulledEnvs, nil
}

func printPullSummary(appId string, pulledEnvs []string) {
	cprint.PrintHeader("--- Pull Summary ---")
	cprint.Success("Successfully pulled and decrypted environment variables")
	cprint.PrintDetail("Application", appId)
	cprint.PrintDetail("Environments pulled", fmt.Sprintf("%d", len(pulledEnvs)))

	cprint.Print("") // Add an empty line for better spacing
	cprint.Print("Pulled environments:")
	for _, env := range pulledEnvs {
		cprint.Print(fmt.Sprintf("  - %s", env))
	}

	cprint.Print("") // Add an empty line for better spacing
	cprint.GreenPrint("All environment variables are now locally available and ready for use.")
}
