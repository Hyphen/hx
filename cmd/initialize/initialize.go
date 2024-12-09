package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var appIDFlag string
var printer *cprint.CPrinter

var InitCmd = &cobra.Command{
	Use:   "init <app name>",
	Short: "Initialize a new Hyphen application in the current directory",
	Long: `
The init command sets up a new Hyphen application in your current directory.

This command will:
- Create a new application in Hyphen
- Generate a local configuration file
- Set up environment files for each project environment
- Update .gitignore to exclude sensitive files

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
	Run:  runInit,
}

func init() {
	InitCmd.Flags().StringVarP(&appIDFlag, "id", "i", "", "App ID (optional)")
}

func runInit(cmd *cobra.Command, args []string) {
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

	appName := ""
	if len(args) == 0 {
		// get the local directory name and prompt if we wish to use this as the app name
		cwd, err := os.Getwd()
		if err != nil {
			printer.Error(cmd, err)
			return
		}

		appName = filepath.Base(cwd)
		response := prompt.PromptYesNo(cmd, fmt.Sprintf("Use the current directory name '%s' as the app name?", appName), true)
		if !response.Confirmed {
			if response.IsFlag {
				printer.Info("Operation cancelled due to --no flag.")
				return
			} else {
				// Prompt for a new app name
				var err error
				appName, err = prompt.PromptString(cmd, "What would you like the app name to be?")
				if err != nil {
					printer.Error(cmd, err)
					return
				}
			}
		}
	} else {
		appName = args[0]
	}

	appAlternateId := getAppID(cmd, appName)
	if appAlternateId == "" {
		return
	}

	if manifest.ExistsLocal() {
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

	mc, err := manifest.RestoreConfig()
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	if mc.ProjectId == nil {
		printer.Error(cmd, fmt.Errorf("No project found in .hx file"))
		return
	}

	newApp, err := appService.CreateApp(orgID, *mc.ProjectId, appAlternateId, appName)
	if err != nil {
		if !errors.Is(err, errors.ErrConflict) {
			printer.Error(cmd, err)
			return
		}

		existingApp, handleErr := handleExistingApp(cmd, *appService, orgID, appAlternateId)
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

	mcl := manifest.Config{
		ProjectId:          mc.ProjectId,
		ProjectAlternateId: mc.ProjectAlternateId,
		ProjectName:        mc.ProjectName,
		OrganizationId:     mc.OrganizationId,
		AppName:            &newApp.Name,
		AppAlternateId:     &newApp.AlternateId,
		AppId:              &newApp.ID,
	}

	ml, err := manifest.LocalInitialize(mcl)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	if err := ensureGitignore(manifest.ManifestSecretFile); err != nil {
		printer.Error(cmd, fmt.Errorf("error adding .hxkey to .gitignore: %w. Please do this manually if you wish", err))
	}

	// List the environments for the project
	environments, err := envService.ListEnvironments(orgID, *ml.ProjectId, 100, 1)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	// Create an empty .env file for each environment, push it up, and add it to .gitignore
	for _, e := range environments {
		envName := strings.ToLower(e.Name)
		envID := e.ID
		err = createAndPushEmptyEnvFile(cmd, envService, ml, orgID, newApp.ID, envID, envName)
		if err != nil {
			printer.Error(cmd, err)
			return
		}
	}

	err = createAndPushEmptyEnvFile(cmd, envService, ml, orgID, newApp.ID, "default", "default")
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	err = createGitignoredFile(cmd, ".env.local")
	if err != nil {
		return
	}

	printInitializationSummary(newApp.Name, newApp.AlternateId, newApp.ID, orgID)
}

func createAndPushEmptyEnvFile(cmd *cobra.Command, envService *env.EnvService, m manifest.Manifest, orgID, appID, envID, envName string) error {
	envFileName, err := env.GetFileName(envName)
	if err != nil {
		return err
	}

	err = createGitignoredFile(cmd, envFileName)
	if err != nil {
		return err
	}

	// Build an Env struct from that new empty file
	envStruct, err := env.GetLocalEncryptedEnv(envName, m)
	if err != nil {
		return err
	}

	version := 1
	envStruct.Version = &version

	if err := envService.PutEnvironmentEnv(orgID, appID, envID, m.SecretKeyId, envStruct); err != nil {
		if !errors.Is(err, errors.ErrConflict) {
			return err
		}
		envStruct, err = envService.GetEnvironmentEnv(orgID, appID, envID, &m.SecretKeyId, nil)
		if err != nil {
			return err
		}
		version = *envStruct.Version

	}

	db, err := database.Restore()
	if err != nil {
		return err
	}

	newEnvDcrypted, err := envStruct.DecryptData(m.GetSecretKey())
	if err != nil {
		return err
	}

	secretKey := database.SecretKey{
		ProjectId: *m.ProjectId,
		AppId:     *m.AppId,
		EnvName:   envName,
	}

	if err := db.UpsertSecret(secretKey, newEnvDcrypted, version); err != nil {
		return fmt.Errorf("failed to save local environment: %w", err) // TODO: check if this should be and error
	}

	return nil
}

func createGitignoredFile(cmd *cobra.Command, fileName string) error {
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

	// Write '# KEY=Value' to the file
	_, err = file.WriteString("# Example\n# KEY=Value\n")
	if err != nil {
		printer.Error(cmd, fmt.Errorf("error writing to %s: %w", fileName, err))
		return err
	}

	if err := ensureGitignore(fileName); err != nil {
		printer.Error(cmd, fmt.Errorf("error adding %s to .gitignore: %w. Please do this manually if you wish", fileName, err))
		// don't error here. Keep going.
	}

	return nil
}

func getAppID(cmd *cobra.Command, appName string) string {
	defaultAppAlternateId := generateDefaultAppId(appName)
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

func generateDefaultAppId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}

func printInitializationSummary(appName, appAlternateId, appID, orgID string) {
	printer.Success("App successfully initialized")
	printer.Print("") // Print an empty line for spacing
	printer.PrintDetail("App Name", appName)
	printer.PrintDetail("App AlternateId", appAlternateId)
	printer.PrintDetail("App ID", appID)
	printer.PrintDetail("Organization ID", orgID)
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

func handleExistingApp(cmd *cobra.Command, appService app.AppService, orgID, appAlternateId string) (*app.App, error) {
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
