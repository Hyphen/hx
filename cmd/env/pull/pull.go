package pull

import (
	"fmt"
	"os"
	"sync"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/secret"
	"github.com/Hyphen/cli/internal/timing"
	"github.com/Hyphen/cli/internal/vinz"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/spf13/cobra"
)

var (
	Silent     bool = false
	forceFlag  bool
	version    int
	versionPtr *int = nil
	printer    *cprint.CPrinter
)

const maxConcurrentEnvOps = 4

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
		versionPtr = nil
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
	recorder := timing.NewRecorder()
	defer recorder.Print(printer, "env pull")

	var cfg config.Config
	if err := recorder.Measure("config load", func() error {
		var err error
		cfg, err = config.RestoreConfig()
		return err
	}); err != nil {
		return err
	}

	// Check if this is a monorepo
	if cfg.IsMonorepoProject() && cfg.Project != nil {
		// Store current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Pull for each workspace app
		for _, appDir := range cfg.Project.Apps {
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
			err = pullForApp(args, forceFlag, recorder)
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

	// If not a monorepo, proceed with regular pull for top-level app
	return pullForApp(args, forceFlag, recorder)
}

func pullForApp(args []string, forceFlag bool, recorder *timing.Recorder) error {
	var db database.Database
	if err := recorder.Measure("database load", func() error {
		var err error
		db, err = database.Restore()
		return err
	}); err != nil {
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

	var secretValue models.Secret
	if err := recorder.Measure("secret load", func() error {
		var err error
		secretValue, _, err = secret.LoadSecret(config.OrganizationId, *config.ProjectId)
		return err
	}); err != nil {
		return err
	}

	switch envName {
	case "": // ALL
		pulledEnvs, err := service.getAllEnvsAndDecryptIntoFiles(orgId, appId, projectId, secretValue, config, forceFlag, recorder)
		if err != nil {
			return err
		}

		if !Silent {
			printPullSummary(pulledEnvs)
		}
		return nil
	case "default":
		if err = service.saveDecryptedEnvIntoFile(orgId, "default", appId, secretValue, config, forceFlag, nil, recorder); err != nil {
			return err
		}

		if !Silent {
			printPullSummary([]string{"default"})
		}
		return nil
	default: // we have a specific env name
		err = service.checkForEnvironment(orgId, envName, projectId)
		if err != nil {
			return err
		}
		if err = service.saveDecryptedEnvIntoFile(orgId, envName, appId, secretValue, config, forceFlag, nil, recorder); err != nil {
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
		return fmt.Errorf("environment %s not found", envName)
	}
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}

	return nil
}

type pullEnvResult struct {
	index     int
	envName   string
	fileName  string
	update    database.SecretUpdate
	hasUpdate bool
	err       error
}

func (s *service) saveDecryptedEnvIntoFile(orgId, envName, appId string, secret models.Secret, cfg config.Config, force bool, listedEnv *models.Env, recorder *timing.Recorder) error {
	var result pullEnvResult
	if err := recorder.Measure("transfer work", func() error {
		result = s.pullEnv(orgId, envName, appId, secret, cfg, force, listedEnv)
		return result.err
	}); err != nil {
		return err
	}

	if result.hasUpdate {
		if err := recorder.Measure("db update", func() error {
			return s.db.UpsertSecrets([]database.SecretUpdate{result.update})
		}); err != nil {
			return err
		}
	}

	if result.fileName != "" {
		_ = gitutil.EnsureGitignore(result.fileName)
	}

	return nil
}

func (s *service) pullEnv(orgId, envName, appId string, secret models.Secret, cfg config.Config, force bool, listedEnv *models.Env) pullEnvResult {
	result := pullEnvResult{
		envName: envName,
	}

	envFileName, err := env.GetFileName(envName)
	if err != nil {
		result.err = err
		return result
	}
	result.fileName = envFileName

	_, err = os.Stat(envFileName)
	fileExists := !os.IsNotExist(err)

	if fileExists && !force {
		currentLocal, err := env.New(envFileName)
		if err != nil {
			result.err = err
			return result
		}

		currentLocalSecret, dbSecretExists := s.db.GetSecret(database.SecretKey{
			ProjectId: *cfg.ProjectId,
			AppId:     *cfg.AppId,
			EnvName:   envName,
		})
		if dbSecretExists {
			actual := currentLocal.HashData()
			expectedHash := currentLocalSecret.Hash
			if actual != expectedHash && !force {
				result.err = fmt.Errorf("local \"%s\" environment has been modified. Use --force to overwrite", envName)
				return result
			}
		}
	}

	e, err := s.getEnvPayloadForPull(orgId, appId, envName, secret, listedEnv)
	if err != nil {
		result.err = err
		return result
	}

	if e.Version == nil {
		result.err = fmt.Errorf("remote \"%s\" environment has no version", envName)
		return result
	}

	envDataDecrypted, err := e.DecryptData(secret)
	if err != nil {
		result.err = err
		return result
	}

	if err := os.WriteFile(envFileName, []byte(envDataDecrypted), 0600); err != nil {
		result.err = fmt.Errorf("failed to save decrypted environment %s to file %s: %w", envName, envFileName, err)
		return result
	}

	result.update = database.SecretUpdate{
		Key: database.SecretKey{
			ProjectId: *cfg.ProjectId,
			AppId:     *cfg.AppId,
			EnvName:   envName,
		},
		Data:    envDataDecrypted,
		Version: *e.Version,
	}
	result.hasUpdate = true

	return result
}

func (s *service) getEnvPayloadForPull(orgId, appId, envName string, secret models.Secret, listedEnv *models.Env) (models.Env, error) {
	if versionPtr == nil && listedEnv != nil && listedEnv.Data != "" && listedEnv.Version != nil &&
		listedEnv.SecretKeyID != nil && *listedEnv.SecretKeyID == secret.SecretKeyId {
		return *listedEnv, nil
	}

	e, err := s.envService.GetEnvironmentEnv(orgId, appId, envName, &secret.SecretKeyId, versionPtr)
	if err == nil {
		return e, nil
	}

	if versionPtr == nil || !errors.Is(err, errors.ErrNotFound) {
		return models.Env{}, err
	}

	if !Silent {
		printer.Warning(fmt.Sprintf("No version found for environment %s. Pulling the latest version.", envName))
	}

	return s.envService.GetEnvironmentEnv(orgId, appId, envName, &secret.SecretKeyId, nil)
}

func (s *service) getAllEnvsAndDecryptIntoFiles(orgId, appId, projectId string, secret models.Secret, cfg config.Config, force bool, recorder *timing.Recorder) ([]string, error) {
	var allEnvs []models.Env
	var currentEnvironments []models.Environment

	if err := recorder.Measure("remote list", func() error {
		var wg sync.WaitGroup
		var allErr error
		var currentErr error

		wg.Add(2)
		go func() {
			defer wg.Done()
			// Currently, api/organizations/:orgId/dot-envs returns all stored ENV files, even if the environment has been deleted.
			allEnvs, allErr = s.envService.ListEnvs(orgId, appId, 100, 1)
		}()
		go func() {
			defer wg.Done()
			// Get the current list of environments that doesn't include deleted ones.
			currentEnvironments, currentErr = s.envService.ListEnvironments(orgId, projectId, 100, 1)
		}()
		wg.Wait()

		if allErr != nil {
			return allErr
		}
		return currentErr
	}); err != nil {
		return nil, err
	}

	// Filters out the environments that have been deleted
	envsSansDeleted := filterEnvsByCurrentEnvironments(allEnvs, currentEnvironments)
	envsByName := envMapByName(allEnvs)

	var results []pullEnvResult
	if err := recorder.Measure("transfer work", func() error {
		results = s.pullEnvsConcurrently(orgId, appId, secret, cfg, force, envsSansDeleted, envsByName)
		return nil
	}); err != nil {
		return nil, err
	}

	var (
		pulledEnvs []string
		updates    []database.SecretUpdate
		files      []string
	)
	for _, result := range results {
		if result.err != nil {
			if !Silent {
				printer.Warning(fmt.Sprintf("Failed to pull environment %s: %s", result.envName, result.err))
			}
			continue
		}
		pulledEnvs = append(pulledEnvs, result.envName)
		if result.hasUpdate {
			updates = append(updates, result.update)
		}
		if result.fileName != "" {
			files = append(files, result.fileName)
		}
	}

	if len(updates) > 0 {
		if err := recorder.Measure("db update", func() error {
			return s.db.UpsertSecrets(updates)
		}); err != nil {
			return nil, err
		}
	}

	// Workaround for apix#1599: Create empty env files for environments that exist in
	// ListEnvironments but not in ListEnvs (new environments with no secrets pushed yet)
	missingEnvs := findMissingEnvironments(allEnvs, currentEnvironments)
	if err := recorder.Measure("local file setup", func() error {
		for _, file := range files {
			_ = gitutil.EnsureGitignore(file)
		}

		for _, envName := range missingEnvs {
			created, err := createEmptyEnvFile(envName)
			if err != nil {
				if !Silent {
					printer.Warning(fmt.Sprintf("Failed to create empty environment file for %s: %s", envName, err))
				}
				continue
			}
			if created {
				printer.PrintVerbose(fmt.Sprintf("Creating empty .env file for missing new environment %s", envName))
				pulledEnvs = append(pulledEnvs, envName)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return pulledEnvs, nil
}

func (s *service) pullEnvsConcurrently(orgId, appId string, secret models.Secret, cfg config.Config, force bool, envNames []string, envsByName map[string]models.Env) []pullEnvResult {
	results := make([]pullEnvResult, len(envNames))
	sem := make(chan struct{}, maxConcurrentEnvOps)
	var wg sync.WaitGroup

	for i, envName := range envNames {
		i := i
		envName := envName
		var listedEnv *models.Env
		if e, ok := envsByName[envName]; ok {
			e := e
			listedEnv = &e
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := s.pullEnv(orgId, envName, appId, secret, cfg, force, listedEnv)
			result.index = i
			results[i] = result
		}()
	}

	wg.Wait()
	return results
}

func envMapByName(allEnvs []models.Env) map[string]models.Env {
	envsByName := make(map[string]models.Env, len(allEnvs))
	for _, e := range allEnvs {
		envName := "default"
		if e.ProjectEnv != nil {
			envName = e.ProjectEnv.AlternateID
		}
		envsByName[envName] = e
	}
	return envsByName
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
// These are new environments that have no secrets pushed yet (see apix#1599).
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
// Returns (created bool, err error) - created is true if file was created, false if it already existed.
func createEmptyEnvFile(envName string) (bool, error) {
	envFileName, err := env.GetFileName(envName)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(envFileName)
	fileExists := !os.IsNotExist(err)

	if fileExists {
		// attempt to add this to .gitignore for the user. Do not fail out if we can't
		_ = gitutil.EnsureGitignore(envFileName)

		// File already exists, skip without error
		return false, nil
	}

	err = os.WriteFile(envFileName, []byte(env.DEFAULT_ENV_CONTENTS), 0644)
	if err != nil {
		return false, err
	}

	// attempt to add this to .gitignore for the user. Do not fail out if we can't
	_ = gitutil.EnsureGitignore(envFileName)

	return true, nil
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
