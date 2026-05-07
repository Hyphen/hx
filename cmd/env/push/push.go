package push

import (
	"fmt"
	"os"
	"strings"
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
	"github.com/spf13/cobra"
)

var Silent bool = false
var printer *cprint.CPrinter

const maxConcurrentEnvOps = 4

var PushCmd = &cobra.Command{
	Use:   "push [environment]",
	Short: "Push local environment variables to Hyphen",
	Long: `
The push command uploads local environment variables from .env files to Hyphen.

This command allows you to:
-  Push all environments found in local .env files when no environment is specified
-  Push a specific environment by name
-  Encrypt and securely store your environment variables in Hyphen

The command looks for .env files in the current directory with the naming convention .env.[environment_name].

Examples:
  hyphen push production
  hyphen push

After pushing, all environment variables will be securely stored in Hyphen and available for use across your project.
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		if err := RunPush(args, cmd); err != nil {
			return err
		}
		return nil
	},
}

func RunPush(args []string, cmd *cobra.Command) error {
	recorder := timing.NewRecorder()
	defer recorder.Print(printer, "env push")

	var cfg config.Config
	if err := recorder.Measure("config load", func() error {
		var err error
		cfg, err = config.RestoreConfig()
		return err
	}); err != nil {
		return err
	}

	var secretValue models.Secret
	if err := recorder.Measure("secret load", func() error {
		var err error
		secretValue, _, err = secret.LoadSecret(cfg.OrganizationId, *cfg.ProjectId)
		return err
	}); err != nil {
		return err
	}

	return runPushUsingSecret(args, secretValue, cmd, recorder)
}

func RunPushUsingSecret(args []string, secret models.Secret, cmd *cobra.Command) error {
	recorder := timing.NewRecorder()
	defer recorder.Print(printer, "env push")

	return runPushUsingSecret(args, secret, cmd, recorder)
}

func runPushUsingSecret(args []string, secret models.Secret, cmd *cobra.Command, recorder *timing.Recorder) error {
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

		// Push for each workspace member
		for _, memberDir := range cfg.Project.Apps {
			if !Silent {
				printer.Print(fmt.Sprintf("Pushing for workspace member: %s", memberDir))
			}

			// Change to member directory
			err = os.Chdir(memberDir)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to change to directory %s: %s", memberDir, err))
				continue
			}

			// Run push for this member
			err = pushForMember(args, secret, cmd, recorder)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to push for member %s: %s", memberDir, err))
			}

			// Change back to original directory
			err = os.Chdir(currentDir)
			if err != nil {
				return fmt.Errorf("failed to return to original directory: %w", err)
			}
		}

		return nil
	}

	// If not a monorepo, proceed with regular push
	return pushForMember(args, secret, cmd, recorder)
}

// pushForMember contains the original push logic
func pushForMember(args []string, secret models.Secret, cmd *cobra.Command, recorder *timing.Recorder) error {
	var cfg config.Config
	if err := recorder.Measure("config load", func() error {
		var err error
		cfg, err = config.RestoreConfig()
		return err
	}); err != nil {
		return err
	}

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

	projectId, err := flags.GetProjectID()
	if err != nil {
		return err
	}

	appId, err := flags.GetApplicationID()
	if err != nil {
		return err
	}

	var envsToPush []string
	var envsPushed []string
	var skippedEnvs []string
	if len(args) == 1 {
		envsToPush = append(envsToPush, args[0])
	} else {
		envsToPush, err = service.getLocalEnvsNamesFromFiles()
		if err != nil {
			return err
		}
	}

	cloudEnvs, err := service.loadPushRemoteState(envsToPush, orgId, appId, projectId, recorder)
	if err != nil {
		return err
	}

	var results []pushEnvResult
	if err := recorder.Measure("transfer work", func() error {
		results = service.pushEnvsConcurrently(orgId, appId, secret, cfg, envsToPush, cloudEnvs)
		return nil
	}); err != nil {
		return err
	}

	var updates []database.SecretUpdate
	for _, result := range results {
		if result.err != nil {
			printer.Error(cmd, result.err)
			continue
		}
		if result.skipped {
			skippedEnvs = append(skippedEnvs, result.envName)
			continue
		}
		envsPushed = append(envsPushed, result.envName)
		if result.hasUpdate {
			updates = append(updates, result.update)
		}
	}

	if len(updates) > 0 {
		if err := recorder.Measure("db update", func() error {
			return service.db.UpsertSecrets(updates)
		}); err != nil {
			return err
		}
	}

	if !Silent {
		printPushSummary(envsToPush, envsPushed, skippedEnvs)
	}
	return nil
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

type pushEnvResult struct {
	index     int
	envName   string
	skipped   bool
	update    database.SecretUpdate
	hasUpdate bool
	err       error
}

func (s *service) pushEnvsConcurrently(orgID, appID string, currentSecret models.Secret, config config.Config, envNames []string, cloudEnvs map[string]models.Env) []pushEnvResult {
	results := make([]pushEnvResult, len(envNames))
	sem := make(chan struct{}, maxConcurrentEnvOps)
	var wg sync.WaitGroup

	for i, envName := range envNames {
		i := i
		envName := envName
		var cloudEnv *models.Env
		if e, ok := cloudEnvs[envName]; ok {
			e := e
			cloudEnv = &e
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := s.pushEnv(orgID, envName, appID, currentSecret, config, cloudEnv)
			result.index = i
			results[i] = result
		}()
	}

	wg.Wait()
	return results
}

func (s *service) pushEnv(orgID, envName, appID string, currentSecret models.Secret, config config.Config, cloudEnv *models.Env) pushEnvResult {
	result := pushEnvResult{
		envName: envName,
	}

	envFileName, err := env.GetFileName(envName)
	if err != nil {
		result.err = err
		return result
	}

	localEnv, err := env.New(envFileName)
	if err != nil {
		result.err = err
		return result
	}
	plainData := localEnv.Data

	// Check local environment
	currentLocalEnv, exists := s.db.GetSecret(database.SecretKey{
		ProjectId: *config.ProjectId,
		AppId:     *config.AppId,
		EnvName:   envName,
	})

	latestCloudEnv, cloudExists, err := s.resolveCloudEnvForPush(orgID, appID, envName, cloudEnv)
	if err != nil {
		result.err = err
		return result
	}

	replacingSecretKeyID := currentSecret.SecretKeyId
	if cloudExists && latestCloudEnv.SecretKeyID != nil {
		replacingSecretKeyID = *latestCloudEnv.SecretKeyID
	}

	if exists && cloudExists && currentLocalEnv.Hash == localEnv.HashData() && currentSecret.SecretKeyId == replacingSecretKeyID {
		result.skipped = true
		return result
	}

	// try pushing version+1
	newVersion := nextEnvVersion(currentLocalEnv, exists, latestCloudEnv, cloudExists)
	localEnv.Version = &newVersion
	localEnv.SecretKeyID = &currentSecret.SecretKeyId

	envEncryptedData, err := localEnv.EncryptData(currentSecret)
	if err != nil {
		result.err = err
		return result
	}
	localEnv.Data = envEncryptedData

	// Update cloud environment
	if err := s.envService.PutEnvironmentEnv(orgID, appID, envName, replacingSecretKeyID, localEnv); err != nil {
		// Workaround for apix#1599: API returns 400 "secretKeyId must be >= 1" when secretKeyId is 0.
		// This happens for new environments. Retry with the project's secret key.
		if strings.Contains(err.Error(), "secretKeyId must be >= 1") {
			if retryErr := s.envService.PutEnvironmentEnv(orgID, appID, envName, currentSecret.SecretKeyId, localEnv); retryErr != nil {
				result.err = fmt.Errorf("failed to update cloud %s environment: %w", envName, retryErr)
				return result
			}
		} else {
			result.err = fmt.Errorf("failed to update cloud %s environment: %w", envName, err)
			return result
		}
	}

	result.update = database.SecretUpdate{
		Key: database.SecretKey{
			ProjectId: *config.ProjectId,
			AppId:     *config.AppId,
			EnvName:   envName,
		},
		Data:    plainData,
		Version: newVersion,
	}
	result.hasUpdate = true

	return result
}

func (s *service) resolveCloudEnvForPush(orgID, appID, envName string, cloudEnv *models.Env) (models.Env, bool, error) {
	if cloudEnv == nil {
		return models.Env{}, false, nil
	}
	if cloudEnv.SecretKeyID != nil {
		return *cloudEnv, true, nil
	}

	latestEnv, err := s.envService.GetEnvironmentEnv(orgID, appID, envName, nil, nil)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return models.Env{}, false, nil
		}
		return models.Env{}, false, err
	}

	return latestEnv, true, nil
}

func nextEnvVersion(local database.Secret, localExists bool, cloud models.Env, cloudExists bool) int {
	if localExists && local.Version > 0 {
		return local.Version + 1
	}

	if cloudExists && cloud.Version != nil && *cloud.Version > 0 {
		return *cloud.Version + 1
	}

	return 1
}

func (s *service) loadPushRemoteState(envs []string, orgId, appId, projectId string, recorder *timing.Recorder) (map[string]models.Env, error) {
	var environments []models.Environment
	var cloudEnvs []models.Env

	if err := recorder.Measure("remote list", func() error {
		var wg sync.WaitGroup
		var environmentsErr error
		var cloudEnvsErr error

		wg.Add(2)
		go func() {
			defer wg.Done()
			environments, environmentsErr = s.envService.ListEnvironments(orgId, projectId, 100, 1)
		}()
		go func() {
			defer wg.Done()
			cloudEnvs, cloudEnvsErr = s.envService.ListEnvs(orgId, appId, 100, 1)
		}()
		wg.Wait()

		if environmentsErr != nil {
			return environmentsErr
		}
		return cloudEnvsErr
	}); err != nil {
		return nil, err
	}

	if err := validateLocalEnvsExistAsEnvironments(envs, environments); err != nil {
		return nil, err
	}

	return pushEnvMapByName(cloudEnvs), nil
}

func pushEnvMapByName(allEnvs []models.Env) map[string]models.Env {
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

func (s *service) checkIfLocalEnvsExistAsEnvironments(envs []string, orgId, projectId string) error {
	environments, err := s.envService.ListEnvironments(orgId, projectId, 100, 1)
	if err != nil {
		return err
	}
	return validateLocalEnvsExistAsEnvironments(envs, environments)
}

func validateLocalEnvsExistAsEnvironments(envs []string, environments []models.Environment) error {
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

func printPushSummary(envsToPush []string, envsPushed []string, skippedEnvs []string) {
	if len(envsToPush) > 1 {
		if len(envsPushed) > 0 {
			printer.Success(fmt.Sprintf("%s %s", "pushed: ", strings.Join(envsPushed, ", ")))
		} else {
			printer.Success("pushed: everything is up to date")
		}
		if flags.VerboseFlag {
			if len(skippedEnvs) > 0 {
				printer.PrintDetail("skipped", strings.Join(skippedEnvs, ", "))
			} else {
				printer.PrintDetail("skipped", "None")
			}
		}
	} else {
		if len(envsToPush) == 1 && len(envsPushed) == 1 {
			printer.Success(fmt.Sprintf("Successfully pushed environment '%s'", envsToPush[0]))
		}
	}

}
