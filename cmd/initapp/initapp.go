package initapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/cmd/entrypoint"
	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	hyphenapp "github.com/Hyphen/cli/internal/hyphenApp"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/secret"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var appIDFlag string
var printer *cprint.CPrinter

var InitCmd = &cobra.Command{
	Use:   "init-app <app name>",
	Short: "Initialize a new Hyphen application in the current directory",
	Long: `
The init command sets up a new Hyphen application in your current directory.

This command will:
- Create a new application in Hyphen
- Generate a local configuration file
- Set up environment files for each project environment
- Update .gitignore to exclude sensitive files
- Create hyphen-entrypoint.sh (intended for Dockerfile usage)

If no app name is provided, it will prompt to use the current directory name.

The command will guide you through:
- Confirming or entering an application name
- Generating or providing an app ID
- Creating necessary local files

After initialization, you'll receive a summary of the new application, including its name,
ID, and associated organization.

Examples:
  hyphen init
  hyphen init "My New App"
  hyphen init "My New App" --id my-custom-app-id
`,
	Args: cobra.MaximumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	Run: func(cmd *cobra.Command, args []string) {
		RunInitApp(cmd, args)
	},
}

func init() {
	printer = cprint.NewCPrinter(flags.VerboseFlag)
	InitCmd.Flags().StringVarP(&appIDFlag, "id", "i", "", "App ID (optional)")
}

func RunInitApp(cmd *cobra.Command, args []string) {
	printer = cprint.NewCPrinter(flags.VerboseFlag)

	if err := isValidDirectory(cmd); err != nil {
		printer.Error(cmd, err)
		printer.Info("Please change to a project directory and try again.")
		return
	}

	appService := app.NewService()
	envService := env.NewService()

	orgID, err := flags.GetOrganizationID()
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	appName, shouldContinue, err := GetAppName(cmd, args)
	if err != nil {
		printer.Error(cmd, err)
		return
	}
	//If the operation is canceled
	if !shouldContinue {
		return
	}

	appAlternateId := GetAppID(cmd, appName)
	if appAlternateId == "" {
		return
	}

	if config.ExistsLocal() {
		response := prompt.PromptYesNo(cmd, "Config file exists. Do you want to overwrite it?", false)
		if !response.Confirmed {
			if response.IsFlag {
				printer.Info("Operation cancelled due to --no flag.")
			} else {
				printer.Info("Operation cancelled.")
			}
			return
		}
	}

	projectID, err := flags.GetProjectID()
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	newApp, err := appService.CreateApp(orgID, projectID, appAlternateId, appName)
	if err != nil {
		if !errors.Is(err, errors.ErrConflict) {
			printer.Error(cmd, err)
			return
		}

		existingApp, handleErr := HandleExistingApp(cmd, *appService, orgID, appAlternateId)
		if handleErr != nil {
			printer.Error(cmd, handleErr)
			return
		}
		if existingApp == nil {
			printer.Info("Operation cancelled.")
			return
		}

		newApp = *existingApp
	}

	mcl := config.Config{
		ProjectId:      &projectID,
		OrganizationId: orgID,
		AppName:        &newApp.Name,
		AppAlternateId: &newApp.AlternateId,
		AppId:          &newApp.ID,
	}

	err = config.InitializeConfig(mcl, config.ManifestConfigFile)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	ms, _, err := secret.LoadOrInitializeSecret(mcl.OrganizationId, *mcl.ProjectId)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	if err := gitutil.EnsureGitignore(secret.ManifestSecretFile); err != nil {
		printer.Error(cmd, fmt.Errorf("error adding .hxkey to .gitignore: %w. Please do this manually if you wish", err))
	}

	// List the environments for the project
	environments, err := envService.ListEnvironments(orgID, *mcl.ProjectId, 100, 1)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	// Create an empty .env file for each environment, push it up, and add it to .gitignore
	for _, e := range environments {
		envName := strings.ToLower(e.Name)
		envID := e.ID
		err = CreateAndPushEmptyEnvFile(cmd, envService, mcl, ms, orgID, newApp.ID, envID, envName)
		if err != nil {
			printer.Error(cmd, err)
			return
		}
	}

	err = CreateAndPushEmptyEnvFile(cmd, envService, mcl, ms, orgID, newApp.ID, "default", "default")
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	err = CreateGitignoredFile(cmd, ".env.local")
	if err != nil {
		return
	}

	err = entrypoint.CreateEntrypoint(true)
	if err != nil {
		return
	}

	PrintInitializationSummary(newApp.Name, newApp.AlternateId, newApp.ID, orgID, projectID)
}

func GetAppName(cmd *cobra.Command, args []string) (string, bool, error) {
	if len(args) > 0 {
		return args[0], true, nil
	}

	// Get the local directory name
	cwd, err := os.Getwd()
	if err != nil {
		return "", false, err
	}

	dirName := filepath.Base(cwd)
	response := prompt.PromptYesNo(cmd, fmt.Sprintf("Use the current directory name '%s' as the app name?", dirName), true)

	if !response.Confirmed {
		if response.IsFlag {
			printer.Info("Operation cancelled due to --no flag.")
			return "", false, nil
		}

		// Prompt for a new app name
		appName, err := prompt.PromptString(cmd, "What would you like the app name to be?")
		if err != nil {
			return "", false, err
		}
		return appName, true, nil
	}

	return dirName, true, nil
}

func CreateAndPushEmptyEnvFile(cmd *cobra.Command, envService *env.EnvService, c config.Config, s models.Secret, orgID, appID, envID, envName string) error {
	envFileName, err := env.GetFileName(envName)
	if err != nil {
		return err
	}

	err = CreateGitignoredFile(cmd, envFileName)
	if err != nil {
		return err
	}

	// Build an Env struct from that new empty file
	envStruct, err := env.GetLocalEncryptedEnv(envName, nil, s)
	if err != nil {
		return err
	}

	version := 1
	envStruct.Version = &version

	if err := envService.PutEnvironmentEnv(orgID, appID, envID, s.SecretKeyId, envStruct); err != nil {
		//if its conflic it means it already exists so me can pull it
		if !errors.Is(err, errors.ErrConflict) {
			return err
		}
		envStruct, err = envService.GetEnvironmentEnv(orgID, appID, envID, &s.SecretKeyId, nil)
		if err != nil {
			return err
		}
		version = *envStruct.Version

	}

	db, err := database.Restore()
	if err != nil {
		return err
	}

	newEnvDecrypted, err := envStruct.DecryptData(s)
	if err != nil {
		return err
	}

	secretKey := database.SecretKey{
		ProjectId: *c.ProjectId,
		AppId:     *c.AppId,
		EnvName:   envName,
	}

	if err := db.UpsertSecret(secretKey, newEnvDecrypted, version); err != nil {
		return fmt.Errorf("failed to save local environment: %w", err)
	}

	return nil
}

func CreateGitignoredFile(cmd *cobra.Command, fileName string) error {
	// check if the file already exists.
	if _, err := os.Stat(fileName); err == nil {
		// do not recreate. file exists already.
		return nil
	}

	// Create the file
	file, err := os.Create(fileName)
	if err != nil {
		printer.Error(cmd, fmt.Errorf("error creating %s: %w", fileName, err))
		return err
	}
	defer file.Close()

	// Write default env contents to the file
	_, err = file.WriteString(env.DEFAULT_ENV_CONTENTS)
	if err != nil {
		printer.Error(cmd, fmt.Errorf("error writing to %s: %w", fileName, err))
		return err
	}

	if err := gitutil.EnsureGitignore(fileName); err != nil {
		printer.Error(cmd, fmt.Errorf("error adding %s to .gitignore: %w. Please do this manually if you wish", fileName, err))
		// don't error here. Keep going.
	}

	return nil
}

func GetAppID(cmd *cobra.Command, appName string) string {
	defaultAppAlternateId := GenerateDefaultAppId(appName)
	appAlternateId := appIDFlag
	if appAlternateId == "" {
		appAlternateId = defaultAppAlternateId
	}

	err := app.CheckAppId(appAlternateId)
	if err != nil {
		suggestedID := strings.TrimSpace(strings.Split(err.Error(), ":")[1])
		response := prompt.PromptYesNo(cmd, fmt.Sprintf("Invalid app ID. Do you want to use the suggested ID [%s]?", suggestedID), true)

		if !response.Confirmed {
			if response.IsFlag {
				printer.Info("Operation cancelled due to --no flag.")
				return ""
			} else {
				// Prompt for a custom app ID
				for {
					var err error
					appAlternateId, err = prompt.PromptString(cmd, "Enter a custom app ID:")
					if err != nil {
						printer.Error(cmd, err)
						return ""
					}

					err = app.CheckAppId(appAlternateId)
					if err == nil {
						return appAlternateId
					}
					printer.Warning("Invalid app ID. Please try again.")
				}
			}
		} else {
			appAlternateId = suggestedID
		}
	}
	return appAlternateId
}

func GenerateDefaultAppId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}

func PrintInitializationSummary(appName, appAlternateId, appID, orgID, projectAlternateId string) {
	printer.Success(fmt.Sprintf("%s successfully initialized", appName))
	printer.Print("") // Print an empty line for spacing
	printer.Print(hyphenapp.ApplicationLink(orgID, projectAlternateId, appAlternateId))
}

func isValidDirectory(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if cwd == homeDir {
		return fmt.Errorf("initialization in home directory not allowed")
	}

	return nil
}

func HandleExistingApp(cmd *cobra.Command, appService app.AppService, orgID, appAlternateId string) (*models.App, error) {
	response := prompt.PromptYesNo(cmd, fmt.Sprintf("An app with ID '%s' already exists. Do you want to use this existing app?", appAlternateId), true)
	if !response.Confirmed {
		return nil, nil
	}

	existingApp, err := appService.GetApp(orgID, appAlternateId)
	if err != nil {
		return nil, err
	}

	printer.Info(fmt.Sprintf("Using existing app '%s' (%s)", existingApp.Name, existingApp.AlternateId))
	return &existingApp, nil
}
