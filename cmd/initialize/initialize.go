package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var appIDFlag string

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
	appService := app.NewService()
	envService := env.NewService()

	orgID, err := flags.GetOrganizationID()
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	appName := ""
	if len(args) == 0 {
		// get the local directory name and prompt if we wish to use this as the app name
		cwd, err := os.Getwd()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		appName = filepath.Base(cwd)
		response := prompt.PromptYesNo(cmd, fmt.Sprintf("Use the current directory name '%s' as the app name?", appName), true)
		if !response.Confirmed {
			if response.IsFlag {
				cprint.Info("Operation cancelled due to --no flag.")
				return
			} else {
				// Prompt for a new app name
				fmt.Print("What would you like the app name to be? ")
				fmt.Scanln(&appName)
				if appName == "" {
					cprint.Info("No app name provided. Operation cancelled.")
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

	response := prompt.PromptYesNo(cmd, "Config file exists. Do you want to overwrite it?", false)
	if manifest.ExistsLocal() && !response.Confirmed {
		if response.IsFlag {
			cprint.Info("Operation cancelled due to --no flag.")
		} else {
			cprint.Info("Operation cancelled.")
		}
		return
	}

	m, err := manifest.Restore()
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	if m.ProjectId == nil {
		cprint.Error(cmd, fmt.Errorf("no project found in Manifest"))
		return
	}

	newApp, err := appService.CreateApp(orgID, *m.ProjectId, appAlternateId, appName)
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	mcl := manifest.ManifestConfig{
		ProjectId:          m.ProjectId,
		ProjectAlternateId: m.ProjectAlternateId,
		ProjectName:        m.ProjectName,
		OrganizationId:     m.OrganizationId,
		AppName:            &newApp.Name,
		AppAlternateId:     &newApp.AlternateId,
		AppId:              &newApp.ID,
	}

	_, err = manifest.LocalInitialize(mcl)
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	if err := ensureGitignore(manifest.ManifestSecretFile); err != nil {
		cprint.Error(cmd, fmt.Errorf("error adding .hxkey to .gitignore: %w. Please do this manually if you wish", err))
	}

	// List the environments for the project
	environments, err := envService.ListEnvironments(orgID, *m.ProjectId, 100, 1)
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	// Create an empty .env file for each environment, push it up, and add it to .gitignore
	for _, e := range environments {
		envName := strings.ToLower(e.Name)
		envID := e.ID
		err = createAndPushEmptyEnvFile(cmd, envService, m, orgID, newApp.ID, envID, envName)
		if err != nil {
			cprint.Error(cmd, err)
			return
		}
	}

	err = createAndPushEmptyEnvFile(cmd, envService, m, orgID, newApp.ID, "default", "default")
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	err = createGitignoredFile(cmd, ".env.local")
	if err != nil {
		return
	}

	printInitializationSummary(newApp.Name, newApp.AlternateId, newApp.ID, orgID)
}

func createAndPushEmptyEnvFile(cmd *cobra.Command, envService *env.EnvService, manifest manifest.Manifest, orgID, appID, envID, envName string) error {
	envFileName := fmt.Sprintf(".env.%s", envName)

	if envName == "default" {
		envFileName = ".env"
	}

	err := createGitignoredFile(cmd, envFileName)
	if err != nil {
		return err
	}

	// Build an Env struct from that new empty file
	envStruct, err := env.GetLocalEnv(envName, manifest)
	if err != nil {
		return err
	}

	if err := envService.PutEnvironmentEnv(orgID, appID, envID, envStruct); err != nil {
		return err
	}

	return nil
}

func createGitignoredFile(cmd *cobra.Command, fileName string) error {
	// check if the file already exists.
	if _, err := os.Stat(fileName); err == nil {
		// do not recreate. file exists already.
	} else {
		if _, err := os.Create(fileName); err != nil {
			cprint.Error(cmd, fmt.Errorf("error creating %s: %w", fileName, err))
			return err
		}
	}

	if err := ensureGitignore(fileName); err != nil {
		cprint.Error(cmd, fmt.Errorf("error adding %s to .gitignore: %w. Please do this manually if you wish", fileName, err))
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
				cprint.Info("Operation cancelled due to --no flag.")
				return ""
			} else {
				// Prompt for a custom app ID
				for {
					fmt.Print("Enter a custom app ID: ")
					fmt.Scanln(&appAlternateId)
					err := app.CheckAppId(appAlternateId)
					if err == nil {
						return appAlternateId
					}
					cprint.Warning("Invalid app ID. Please try again.")
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
	cprint.Success("App successfully initialized")
	cprint.Print("") // Print an empty line for spacing
	cprint.PrintDetail("App Name", appName)
	cprint.PrintDetail("App AlternateId", appAlternateId)
	cprint.PrintDetail("App ID", appID)
	cprint.PrintDetail("Organization ID", orgID)
}
