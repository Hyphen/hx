package pull

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/secretkey"
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

		// Pull for each workspace member
		for _, memberDir := range manifest.Workspace.Members {
			if !Silent {
				printer.Print(fmt.Sprintf("Pulling for workspace member: %s", memberDir))
			}

			// Change to member directory
			err = os.Chdir(memberDir)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to change to directory %s: %s", memberDir, err))
				continue
			}

			// Run pull for this member
			err = pullForMember(args, forceFlag)
			if err != nil {
				printer.Warning(fmt.Sprintf("Failed to pull for member %s: %s", memberDir, err))
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
	return pullForMember(args, forceFlag)
}

// pullForMember contains the original pull logic
func pullForMember(args []string, forceFlag bool) error {
	db, err := database.Restore()
	if err != nil {
		return err
	}

	service := newService(env.NewService(), db)

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

	manifest, err := manifest.Restore()
	if err != nil {
		return err
	}

	var envName string
	if len(args) == 1 {
		envName = args[0]
	}

	if envName == "" { // ALL
		pulledEnvs, err := service.getAllEnvsAndDecryptIntoFiles(orgId, appId, manifest.GetSecretKey(), manifest, forceFlag)
		if err != nil {
			return err
		}

		if !Silent {
			printPullSummary(pulledEnvs)
		}
		return nil
	} else if envName == "default" {
		if err = service.saveDecryptedEnvIntoFile(orgId, "default", appId, manifest.GetSecretKey(), manifest, forceFlag); err != nil {
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
		if err = service.saveDecryptedEnvIntoFile(orgId, envName, appId, manifest.GetSecretKey(), manifest, forceFlag); err != nil {
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

func (s *service) saveDecryptedEnvIntoFile(orgId, envName, appId string, secretKey *secretkey.SecretKey, m manifest.Manifest, force bool) error {
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
			ProjectId: *m.ProjectId,
			AppId:     *m.AppId,
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

	e, err := s.envService.GetEnvironmentEnv(orgId, appId, envName, &m.SecretKeyId, versionPtr)

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
		e, err = s.envService.GetEnvironmentEnv(orgId, appId, envName, &m.SecretKeyId, nil)
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
			ProjectId: *m.ProjectId,
			AppId:     *m.AppId,
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

func (s *service) getAllEnvsAndDecryptIntoFiles(orgId, appId string, secretkey *secretkey.SecretKey, m manifest.Manifest, force bool) ([]string, error) {
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
		if err := s.saveDecryptedEnvIntoFile(orgId, envName, appId, secretkey, m, force); err != nil {
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
