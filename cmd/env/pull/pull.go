package pull

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/secret"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/internal/vinz"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	Silent     bool = false
	forceFlag  bool
	version    int
	versionPtr *int = nil
	printer    *cprint.CPrinter
)

var PullCmd = &cobra.Command{
	Use:   "pull [environment]",
	Short: "Pull and decrypt environment variables from Hyphen",
	Long: `
The pull command retrieves environment variables from Hyphen and decrypts them into local .env files.

This command allows you to:
- Pull a specific environment by name
- Pull all environments for the application

The pulled environments will be decrypted and saved as .env.[environment_name] files in your current directory.

Examples:
  hyphen pull production
  hyphen pull

After pulling, all environment variables will be locally available and ready for use.
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		if version != 0 {
			versionPtr = &version
		}
		if err := RunPull(args, forceFlag); err != nil {
			printer.Error(cmd, err)
		}
	},
}

func RunPull(args []string, forceFlag bool) error {
	config, err := config.RestoreConfig()
	if err != nil {
		return err
	}

	// Check if this is a monorepo
	if config.IsMonorepoProject() && config.Project != nil {
		// Store current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Pull for each workspace app
		for _, appDir := range config.Project.Apps {
			if !Silent {
				printer.Print(fmt.Sprintf("Pulling for workspace app: %s", appDir))
			}

			// Change to app directory
			err = os.Chdir(appDir)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to change to directory %s: %s", appDir, err))
				continue
			}

			// Run pull for this app
			err = pullForApp(args, forceFlag)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to pull for app %s: %s", appDir, err))
			}

			// Change back to original directory
			err = os.Chdir(currentDir)
			if err != nil {
				return fmt.Errorf("failed to return to original directory: %w", err)
			}
		}

		return nil
	}

	// If not a monorepo, proceed with regular pull
	return pullForApp(args, forceFlag)
}

// pullForApp contains the original pull logic
func pullForApp(args []string, forceFlag bool) error {
	db, err := database.Restore()
	if err != nil {
		return err
	}

	service := newService(env.NewService(), db, vinz.NewService())

	orgId, err := flags.GetOrganizationID()
	if err != nil {
		return err
	}

	appId, err := flags.GetApplicationID()
	if err != nil {
		return err
	}

	projectId, err := flags.GetProjectID()
	if err != nil {
		return err
	}

	config, err := config.RestoreConfig()
	if err != nil {
		return err
	}

	var envName string
	if len(args) == 1 {
		envName = args[0]
	}

	secret, err := secret.LoadSecret(*&config.OrganizationId, *config.ProjectId, true)
	if err != nil {
		return err
	}
	secretKey := service.getSecretKey(orgId, projectId, secret)

	if envName == "" { // ALL
		pulledEnvs, err := service.getAllEnvsAndDecryptIntoFiles(orgId, appId, secretKey, config, secret, forceFlag)
		if err != nil {
			return err
		}

		if !Silent {
			printPullSummary(pulledEnvs)
		}
		return nil
	} else if envName == "default" {
		if err = service.saveDecryptedEnvIntoFile(orgId, "default", appId, secretKey, config, secret, forceFlag); err != nil {
			return err
		}

		if !Silent {
			printPullSummary([]string{"default"})
		}
		return nil
	} else { // we have a specific env name
		err = service.checkForEnvironment(orgId, envName, projectId)
		if err != nil {
			return err
		}
		if err = service.saveDecryptedEnvIntoFile(orgId, envName, appId, secretKey, config, secret, forceFlag); err != nil {
			return err
		}

		if !Silent {
			printPullSummary([]string{envName})
		}
		return nil
	}
}

func init() {
	PullCmd.Flags().BoolVar(&forceFlag, "force", false, "Force overwrite of locally modified environment files")
	PullCmd.Flags().IntVar(&version, "version", 0, "Specify a version to pull")
}

type service struct {
	envService  env.EnvServicer
	vinzService vinz.VinzServicer
	db          database.Database
}

func newService(envService env.EnvServicer, db database.Database, vinzService vinz.VinzServicer) *service {
	return &service{
		envService,
		vinzService,
		db,
	}
}

func (s *service) getSecretKey(orgId, projectId string, secret models.Secret) *secretkey.SecretKey {
	return secretkey.FromBase64(secret.Base64SecretKey)
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

func (s *service) saveDecryptedEnvIntoFile(orgId, envName, appId string, secretKey *secretkey.SecretKey, config config.Config, secret models.Secret, force bool) error {
	envFileName, err := env.GetFileName(envName)
	if err != nil {
		return err
	}

	_, err = os.Stat(envFileName)
	fileExists := !os.IsNotExist(err)

	if fileExists && !force {
		currentLocal, err := env.New(envFileName)
		if err != nil {
			return err
		}

		currentLocalSecret, dbSecretExists := s.db.GetSecret(database.SecretKey{
			ProjectId: *config.ProjectId,
			AppId:     *config.AppId,
			EnvName:   envName,
		})
		if dbSecretExists {
			actual := currentLocal.HashData()
			expectedHash := currentLocalSecret.Hash
			if actual != expectedHash && !force {
				return fmt.Errorf("Local \"%s\" environment has been modified. Use --force to overwrite", envName)
			}
		}
	}

	e, err := s.envService.GetEnvironmentEnv(orgId, appId, envName, &secret.SecretKeyId, versionPtr)

	// Case 1: Error occurred and no version specified
	if err != nil && versionPtr == nil {
		return err
	}

	// Case 2: Error occurred and version specified
	if err != nil && versionPtr != nil {
		// Check if it's a NotFound error
		if !errors.Is(err, errors.ErrNotFound) {
			return err
		}

		// Handle NotFound error
		if !Silent {
			printer.Warning(fmt.Sprintf("No version found for environment %s. Pulling the latest version.", envName))
		}

		// Retry without version
		e, err = s.envService.GetEnvironmentEnv(orgId, appId, envName, &secret.SecretKeyId, nil)
		if err != nil {
			return err
		}
	}

	envDataDecrypted, err := e.DecryptData(secretKey)
	if err != nil {
		return err
	}

	if err := s.db.UpsertSecret(
		database.SecretKey{
			ProjectId: *config.ProjectId,
			AppId:     *config.AppId,
			EnvName:   envName,
		},
		envDataDecrypted, *e.Version); err != nil {
		return err
	}

	if _, err = e.DecryptVarsAndSaveIntoFile(envFileName, secretKey); err != nil {
		return fmt.Errorf("Failed to save decrypted environment: %s, variables to file: %s", envName, envFileName)
	}

	return nil
}

func (s *service) getAllEnvsAndDecryptIntoFiles(orgId, appId string, secretkey *secretkey.SecretKey, config config.Config, secret models.Secret, force bool) ([]string, error) {
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
		if err := s.saveDecryptedEnvIntoFile(orgId, envName, appId, secretkey, config, secret, force); err != nil {
			if !Silent {
				printer.Warning(fmt.Sprintf("Failed to pull environment %s: %s", envName, err))
			}
			continue
		}
		pulledEnvs = append(pulledEnvs, envName)

	}
	return pulledEnvs, nil
}

func printPullSummary(pulledEnvs []string) {
	if len(pulledEnvs) == 0 {
		printer.Print("No environments pulled")
		return
	}
	printer.Print("Pulled environments:")
	for _, env := range pulledEnvs {
		if env == "default" {
			printer.Print("  - default -> .env")
		} else {
			printer.Print(fmt.Sprintf("  - %s -> .env.%s", env, env))
		}
	}
}
