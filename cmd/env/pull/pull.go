package pull

import (
	"fmt"

	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var PullCmd = &cobra.Command{
	Use:   "pull [environment]",
	Short: "Pull and decrypt environment variables from Hyphen",
	Long: `
The pull command retrieves environment variables from Hyphen and decrypts them into local .env files.

This command allows you to:
- Pull a specific environment by name
- Pull all environments for the application
- Decrypt the pulled variables using your local secret key
- Save the decrypted variables into corresponding .env files

The pulled environments will be saved as .env.[environment_name] files in your current directory.

Examples:
  hyphen pull production
  hyphen pull --all

After pulling, all environment variables will be locally available and ready for use.
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Pulling environment variables...")
		manifest, err := manifest.Restore()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		database, err := database.Restore()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		service := newService(env.NewService(), database)

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

		var envName string
		if len(args) == 1 {
			envName = args[0]
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
			if err = service.saveDecryptedEnvIntoFile(orgId, "default", appId, manifest.GetSecretKey()); err != nil {
				cprint.Error(cmd, err)
				return
			}

			printPullSummary(appId, []string{"default"})
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
	db         database.Database
}

func newService(envService env.EnvServicer, db database.Database) *service {
	return &service{
		envService,
		db,
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
	e, err := s.envService.GetEnvironmentEnv(orgId, appId, envName)
	if err != nil {
		return err
	}

	if err := s.db.SaveSecret(envName, e.Data, *e.Version); err != nil {
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
		envName := "default"
		if e.ProjectEnv != nil {
			envName = e.ProjectEnv.AlternateID
		}
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
		if env == "default" {
			cprint.Print("  - default -> .env")
		} else {
			cprint.Print(fmt.Sprintf("  - %s -> .env.%s", env, env))
		}
	}

	cprint.Print("")
	cprint.GreenPrint("All environment variables are now locally available and ready for use.")
}
