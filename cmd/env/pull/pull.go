package pull

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/secret"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		if version != 0 {
			versionPtr = &version
		}
		if err := RunPull(args, forceFlag); err != nil {
			return err
		}
		return nil
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

	secret, _, err := secret.LoadSecret(config.OrganizationId, *config.ProjectId)
	if err != nil {
		return err
	}

	if envName == "" { // ALL
		pulledEnvs, err := service.getAllEnvsAndDecryptIntoFiles(orgId, appId, projectId, secret, config, forceFlag)
		if err != nil {
			return err
		}

		if !Silent {
			printPullSummary(pulledEnvs)
		}
		return nil
	} else if envName == "default" {
		if err = service.saveDecryptedEnvIntoFile(orgId, "default", appId, secret, config, forceFlag); err != nil {
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
		if err = service.saveDecryptedEnvIntoFile(orgId, envName, appId, secret, config, forceFlag); err != nil {
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

func (s *service) saveDecryptedEnvIntoFile(orgId, envName, appId string, secret models.Secret, config config.Config, force bool) error {
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

	envDataDecrypted, err := e.DecryptData(secret)
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

	if _, err = e.DecryptVarsAndSaveIntoFile(envFileName, secret); err != nil {
		return fmt.Errorf("Failed to save decrypted environment: %s, variables to file: %s", envName, envFileName)
	}

	return nil
}

func (s *service) getAllEnvsAndDecryptIntoFiles(orgId, appId, projectId string, secret models.Secret, config config.Config, force bool) ([]string, error) {
	// Currently, api/organizations/:orgId/dot-envs returns all stored ENV files, even if the environment has been deleted
	allEnvs, err := s.envService.ListEnvs(orgId, appId, 100, 1)
	if err != nil {
		return nil, err
	}

	// Get the current list of environments that doesn't include deleted ones
	currentEnvironments, err := s.envService.ListEnvironments(orgId, projectId, 100, 1)
	if err != nil {
		return nil, err
	}

	// Filters out the environments that have been deleted
	envsSansDeleted := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)

	var pulledEnvs []string
	for _, envName := range envsSansDeleted {
		if err := s.saveDecryptedEnvIntoFile(orgId, envName, appId, secret, config, force); err != nil {
			if !Silent {
				printer.Warning(fmt.Sprintf("Failed to pull environment %s: %s", envName, err))
			}
			continue
		}
		pulledEnvs = append(pulledEnvs, envName)

	}

	// Workaround for apix#1599: Create empty env files for environments that exist in
	// ListEnvironments but not in ListEnvs (new environments with no secrets pushed yet)
	missingEnvs := findMissingEnvironments(allEnvs, currentEnvironments)
	for _, envName := range missingEnvs {
		printer.PrintVerbose(fmt.Sprintf("Creating empty .env file for new environment %s", envName))
		if err := createEmptyEnvFile(envName, force); err != nil {
			if !Silent {
				printer.Warning(fmt.Sprintf("Failed to create empty environment file for %s: %s", envName, err))
			}
			continue
		}
		pulledEnvs = append(pulledEnvs, envName)
	}

	return pulledEnvs, nil
}

func filterEnvsByCurrentEnvironments(allEnvs []models.Env, validEnvironments []models.Environment) []string {
	validEnvNames := make(map[string]bool)
	for _, env := range validEnvironments {
		validEnvNames[env.AlternateID] = true
	}

	var filteredEnvNames []string
	for _, e := range allEnvs {
		envName := "default"
		if e.ProjectEnv != nil {
			envName = e.ProjectEnv.AlternateID
		}

		// Skip environments that no longer exist (except "default" which always exists)
		if envName != "default" && !validEnvNames[envName] {
			if printer != nil {
				printer.PrintVerbose(fmt.Sprintf("Skipping deleted environment: %s", envName))
			}
			continue
		}

		filteredEnvNames = append(filteredEnvNames, envName)
	}

	return filteredEnvNames
}

// findMissingEnvironments returns environments that exist in currentEnvironments but not in allEnvs.
// These are new environments that have no secrets pushed yet.
func findMissingEnvironments(allEnvs []models.Env, currentEnvironments []models.Environment) []string {
	// Build a set of env names that have env files
	existingEnvNames := make(map[string]bool)
	for _, e := range allEnvs {
		envName := "default"
		if e.ProjectEnv != nil {
			envName = e.ProjectEnv.AlternateID
		}
		existingEnvNames[envName] = true
	}

	// Find environments that don't have env files yet
	var missingEnvs []string
	for _, env := range currentEnvironments {
		if !existingEnvNames[env.AlternateID] {
			missingEnvs = append(missingEnvs, env.AlternateID)
		}
	}

	return missingEnvs
}

// createEmptyEnvFile creates an empty .env file for the given environment name.
func createEmptyEnvFile(envName string, force bool) error {
	envFileName, err := env.GetFileName(envName)
	if err != nil {
		return err
	}

	_, err = os.Stat(envFileName)
	fileExists := !os.IsNotExist(err)

	if fileExists && !force {
		return fmt.Errorf("file %s already exists, use --force to overwrite", envFileName)
	}

	return os.WriteFile(envFileName, []byte(""), 0644)
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
