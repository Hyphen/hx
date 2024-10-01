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
	Short: "Initialize a new app",
	Long: `The 'hyphen init' command initializes a new app within your organization.

You need to provide an app name as a positional argument. Optionally, you can specify a custom app ID using the '--id' flag. If no app ID is provided, a default ID will be generated based on the app name.

If a manifest file already exists, you will be prompted to confirm if you want to overwrite it, unless the '--yes' flag is provided, in which case the manifest file will be overwritten automatically.

Example usages:
	hyphen init
  hyphen init MyApp
  hyphen init MyApp --id custom-app-id --yes

Flags:
  --id, -i   Specify a custom app ID (optional)
  --yes, -y  Automatically confirm prompts and overwrite files if necessary`,
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
		if !prompt.PromptYesNo(cmd, fmt.Sprintf("Use the current directory name '%s' as the app name?", appName), true) {
			cprint.Info("Operation cancelled.")
			return
		}
	} else {
		appName = args[0]
	}

	appAlternateId := getAppID(cmd, appName)
	if appAlternateId == "" {
		return
	}

	if manifest.ExistsLocal() && !prompt.PromptYesNo(cmd, "Config file exists. Do you want to overwrite it?", false) {
		cprint.Info("Operation cancelled.")
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
		envFileName := fmt.Sprintf(".env.%s", envName)
		err := createGitignoredFile(cmd, envFileName)
		if err != nil {
			return
		}

		// Build an Env struct from that new empty file
		envStruct, err := env.GetLocalEnv(envName, m)
		if err != nil {
			cprint.Error(cmd, err)
			return
		}

		if err := envService.PutEnv(orgID, newApp.ID, envID, envStruct); err != nil {
			cprint.Error(cmd, err)
			return
		}
	}

	// TODO -- we should actually push this up as an empty default as well.
	err = createGitignoredFile(cmd, ".env")
	if err != nil {
		return
	}
	err = createGitignoredFile(cmd, ".env.local")
	if err != nil {
		return
	}

	printInitializationSummary(newApp.Name, newApp.AlternateId, newApp.ID, orgID)
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
		if !prompt.PromptYesNo(cmd, fmt.Sprintf("Invalid app ID. Do you want to use the suggested ID [%s]?", suggestedID), true) {
			cprint.Info("Operation cancelled.")
			return ""
		}
		appAlternateId = suggestedID
	}
	return appAlternateId
}

func generateDefaultAppId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}

func printInitializationSummary(appName, appAlternateId, appID, orgID string) {
	cprint.PrintHeader("--- App Initialization Summary ---")
	cprint.Success("App successfully initialized")
	cprint.PrintDetail("App Name", appName)
	cprint.PrintDetail("App AlternateId", appAlternateId)
	cprint.PrintDetail("App ID", appID)
	cprint.PrintDetail("Organization ID", orgID)
}
