package pull

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	forceFlag bool
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
		if err := RunPull(args, forceFlag); err != nil {
			cprint.Error(cmd, err)
		}
	},
}

func RunPull(args []string, forceFlag bool) error {
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

	appId, err := flags.GetApplicationID()
	if err != nil {
		return err
	}

	projectId, err := flags.GetProjectID()
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

		printPullSummary(pulledEnvs)
		return nil
	} else if envName == "default" {
		if err = service.saveDecryptedEnvIntoFile(orgId, "default", appId, manifest.GetSecretKey(), manifest, forceFlag); err != nil {
			return err
		}

		printPullSummary([]string{"default"})
		return nil
	} else { // we have a specific env name
		err = service.checkForEnvironment(orgId, envName, projectId)
		if err != nil {
			return err
		}
		if err = service.saveDecryptedEnvIntoFile(orgId, envName, appId, manifest.GetSecretKey(), manifest, forceFlag); err != nil {
			return err
		}

		printPullSummary([]string{envName})
		return nil
	}
}

func init() {
	PullCmd.Flags().BoolVar(&forceFlag, "force", false, "Force overwrite of locally modified environment files")
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

	e, err := s.envService.GetEnvironmentEnv(orgId, appId, envName, m.SecretKeyId)
	if err != nil {
		return err
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
			return pulledEnvs, err
		}
		pulledEnvs = append(pulledEnvs, envName)
	}
	return pulledEnvs, nil
}

func printPullSummary(pulledEnvs []string) {
	cprint.Print("Pulled environments:")
	for _, env := range pulledEnvs {
		if env == "default" {
			cprint.Print("  - default -> .env")
		} else {
			cprint.Print(fmt.Sprintf("  - %s -> .env.%s", env, env))
		}
	}
}
