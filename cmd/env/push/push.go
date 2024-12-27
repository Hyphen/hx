package push

import (
	"fmt"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var Silent bool = false
var printer *cprint.CPrinter

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
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		m, err := manifest.Restore()
		if err != nil {
			printer.Error(cmd, err)
		}
		if err := RunPush(args, m.SecretKeyId); err != nil {
			printer.Error(cmd, err)
		}
	},
}

func RunPush(args []string, secretKeyId int64) error {
	manifest, err := manifest.Restore()
	if err != nil {
		return err
	}

	// Check if this is a monorepo
	if manifest.IsMonorepoProject() && manifest.Workspace != nil {
		// Store current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Push for each workspace member
		for _, memberDir := range manifest.Workspace.Members {
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
			err = pushForMember(args, secretKeyId)
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
	return pushForMember(args, secretKeyId)
}

func pushForMember(args []string, secretKeyId int64) error {
	manifest, err := manifest.Restore()
	if err != nil {
		return err
	}

	db, err := database.Restore()
	if err != nil {
		return err
	}

	service := newService(env.NewService(), db)

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

	if err := service.checkIfLocalEnvsExistAsEnvironments(envsToPush, orgId, projectId); err != nil {
		return err
	}

	for _, envName := range envsToPush {
		e, err := env.GetLocalEncryptedEnv(envName, nil, manifest)
		if err != nil {
			printer.Error(nil, err)
			continue
		}
		err, skippable := service.putEnv(orgId, envName, appId, e, manifest.GetSecretKey(), manifest, secretKeyId, &skippedEnvs)
		if err != nil {
			printer.Error(nil, err)
			continue
		} else if !skippable {
			envsPushed = append(envsPushed, envName)
		}
	}

	if !Silent {
		printPushSummary(envsToPush, envsPushed, skippedEnvs)
	}
	return nil
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

func (s *service) putEnv(orgID, envName, appID string, e env.Env, secretKey secretkey.SecretKeyer, m manifest.Manifest, secretKeyId int64, skippedEnvs *[]string) (err error, skippable bool) {
	// Check local environment
	currentLocalEnv, exists := s.db.GetSecret(database.SecretKey{
		ProjectId: *m.ProjectId,
		AppId:     *m.AppId,
		EnvName:   envName,
	})
	if exists {
		err, skippable := s.validateLocalEnv(&currentLocalEnv, &e, secretKey)
		if err != nil {
			return err, false
		}
		if skippable && m.SecretKeyId == secretKeyId {
			*skippedEnvs = append(*skippedEnvs, envName)
			return nil, true
		}
	}

	// try pushing version+1
	newVersion := currentLocalEnv.Version + 1
	e.Version = &newVersion

	// Update cloud environment
	if err := s.envService.PutEnvironmentEnv(orgID, appID, envName, secretKeyId, e); err != nil {
		return fmt.Errorf("failed to update cloud %s environment: %w", envName, err), false
	}

	// Update local environment
	newEnvDcrypted, err := e.DecryptData(secretKey)
	if err != nil {
		return err, false
	}
	if err := s.db.UpsertSecret(database.SecretKey{
		ProjectId: *m.ProjectId,
		AppId:     *m.AppId,
		EnvName:   envName,
	}, newEnvDcrypted, currentLocalEnv.Version+1); err != nil {
		return fmt.Errorf("failed to save local %s environment: %w", envName, err), false
	}

	return nil, false
}

func (s *service) validateLocalEnv(local *database.Secret, new *env.Env, secretKey secretkey.SecretKeyer) (err error, skippable bool) {
	newEnvDcrypted, err := new.DecryptData(secretKey)
	if err != nil {
		return err, false
	}
	if local.Hash == env.HashData(newEnvDcrypted) {
		return nil, true
	}

	return nil, false
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

func printPushSummary(envsToPush []string, envsPushed []string, skippedEnvs []string) {
	if len(envsToPush) > 1 {
		if len(envsPushed) > 0 {
			printer.Success(fmt.Sprintf("%s %s", "pushed: ", strings.Join(envsPushed, ", ")))
		} else {
			printer.Info("No environments were pushed, everything is up to date.")
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
