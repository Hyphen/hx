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
	Short: "Pull and decrypt environment variables from Hyphen",
	Long: `
The pull command retrieves environment variables from Hyphen and decrypts them into local .env files.

This command allows you to:
- Pull a specific environment by name
- Pull all environments for the application
- Decrypt the pulled variables using your local secret key
- Save the decrypted variables into corresponding .env files

The pulled environments will be saved as .env.[environment_name] files in your current directory.

Note: Pulling the default environment is not yet implemented. Use --all to pull all environments or -e [environment] to pull a specific one.

Examples:
  hyphen pull -e production
  hyphen pull --all

After pulling, all environment variables will be locally available and ready for use.
`,
	Run: func(cmd *cobra.Command, args []string) {
		service := newService(env.NewService())

		orgId, err := flags.GetOrganizationID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		appId, err := flags.GetApplicationID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		projectId, err := flags.GetProjectID()
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
			cprint.Error(cmd, err)
			return
		}

		if flags.AllFlag {
			pulledEnvs, err := service.getAllEnvsAndDecryptIntoFiles(orgId, appId, manifest.GetSecretKey())
			if err != nil {
				cprint.Error(cmd, err)
				return
			}

			printPullSummary(appId, pulledEnvs)
			return
		}

		if envName == "" || envName == "default" {
			// TODO
			// We want to just pull default here, not all.
			cprint.Error(cmd, fmt.Errorf("pulling default environment not yet implemented. Please use --all to pull all, or -e [environment] to pull a specific environment"))
			return
		} else { // we have a specific env name
			err = service.checkForEnvironment(orgId, envName, projectId)
			if err != nil {
				cprint.Error(cmd, err)
				return
			}
			if err = service.saveDecryptedEnvIntoFile(orgId, envName, appId, manifest.GetSecretKey()); err != nil {
				cprint.Error(cmd, err)
				return
			}

			printPullSummary(appId, []string{envName})
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

func (s *service) saveDecryptedEnvIntoFile(orgId, envName, appId string, secretKey *secretkey.SecretKey) error {
	e, err := s.envService.GetEnv(orgId, appId, envName)
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

func (s *service) getAllEnvsAndDecryptIntoFiles(orgId, appId string, secretkey *secretkey.SecretKey) ([]string, error) {
	allEnvs, err := s.envService.ListEnvs(orgId, appId, 100, 1)
	if err != nil {
		return nil, err
	}
	var pulledEnvs []string
	for _, e := range allEnvs {
		envName := e.ProjectEnv.AlternateID
		if err := s.saveDecryptedEnvIntoFile(orgId, envName, appId, secretkey); err != nil {
			return pulledEnvs, err
		}
		pulledEnvs = append(pulledEnvs, envName)
	}
	return pulledEnvs, nil
}

func printPullSummary(appId string, pulledEnvs []string) {
	cprint.Success("Successfully pulled and decrypted environment variables")
	cprint.Print("")
	cprint.PrintDetail("Application", appId)
	cprint.PrintDetail("Environments pulled", fmt.Sprintf("%d", len(pulledEnvs)))

	cprint.Print("")
	cprint.Print("Pulled environments:")
	for _, env := range pulledEnvs {
		cprint.Print(fmt.Sprintf("  - %s -> .env.%s", env, env))
	}

	cprint.Print("")
	cprint.GreenPrint("All environment variables are now locally available and ready for use.")
}
