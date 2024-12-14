package initialize

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
	"go.uber.org/thriftrw/ptr"
)

func runInitMonorepo(cmd *cobra.Command, args []string) {
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

	monorepoAppName, shouldContinue, err := getAppName(cmd, args)
	if err != nil {
		printer.Error(cmd, err)
		return
	}
	if !shouldContinue {
		return
	}

	monorepoAppAlternateId := getAppID(cmd, monorepoAppName)
	if monorepoAppAlternateId == "" {
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

	// Create monorepo app
	newApp, err := appService.CreateApp(orgID, *mc.ProjectId, monorepoAppAlternateId, monorepoAppName, false)
	if err != nil {
		if !errors.Is(err, errors.ErrConflict) {
			printer.Error(cmd, err)
			return
		}

		existingApp, handleErr := handleExistingApp(cmd, *appService, orgID, monorepoAppAlternateId)
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

	// Initialize monorepo config and secret
	mcl := manifest.Config{
		ProjectId:          mc.ProjectId,
		ProjectAlternateId: mc.ProjectAlternateId,
		ProjectName:        mc.ProjectName,
		OrganizationId:     mc.OrganizationId,
		AppName:            &newApp.Name,
		AppAlternateId:     &newApp.AlternateId,
		AppId:              &newApp.ID,
		IsMonorepo:         ptr.Bool(true),
	}

	ml, err := manifest.LocalInitialize(mcl)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	// Get apps to initialize
	appsOfMonorepoDir, err := prompt.PromptForMonorepoApps(cmd)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	// Initialize each app
	for _, appDir := range appsOfMonorepoDir {
		printer.Info(fmt.Sprintf("Initializing app %s", appDir))
		err := initializeMonorepoApp(cmd, appDir, orgID, mcl, appService, envService, ml.Secret)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to initialize app %s: %w", appDir, err))
			continue
		}
	}
}

func initializeMonorepoApp(cmd *cobra.Command, appDir string, orgID string, mc manifest.Config, appService *app.AppService, envService *env.EnvService, monorepoSecret manifest.Secret) error {
	// Get the app name from directory
	appName := filepath.Base(appDir)

	// Generate app ID
	appAlternateId := generateDefaultAppId(appName)

	// Create the app in Hyphen
	newApp, err := appService.CreateApp(orgID, *mc.ProjectId, appAlternateId, appName, false)
	if err != nil {
		if !errors.Is(err, errors.ErrConflict) {
			return err
		}

		existingApp, handleErr := handleExistingApp(cmd, *appService, orgID, appAlternateId)
		if handleErr != nil {
			return handleErr
		}
		if existingApp == nil {
			return fmt.Errorf("operation cancelled for app: %s", appName)
		}

		newApp = *existingApp
	}

	// Create app config
	mcl := manifest.Config{
		ProjectId:          mc.ProjectId,
		ProjectAlternateId: mc.ProjectAlternateId,
		ProjectName:        mc.ProjectName,
		OrganizationId:     mc.OrganizationId,
		AppName:            &newApp.Name,
		AppAlternateId:     &newApp.AlternateId,
		AppId:              &newApp.ID,
	}

	// Initialize only the config file in the app directory
	err = manifest.InitializeConfig(mcl, filepath.Join(appDir, manifest.ManifestConfigFile))
	if err != nil {
		return err
	}

	// Create a manifest with the monorepo secret
	ml := manifest.Manifest{
		Config: mcl,
		Secret: monorepoSecret,
	}

	// List environments for the project
	environments, err := envService.ListEnvironments(orgID, *ml.ProjectId, 100, 1)
	if err != nil {
		return err
	}

	// Create environment files
	for _, e := range environments {
		envName := strings.ToLower(e.Name)
		envID := e.ID
		err = createAndPushEmptyEnvFile(cmd, envService, ml, orgID, newApp.ID, envID, envName)
		if err != nil {
			return err
		}
	}

	err = createAndPushEmptyEnvFile(cmd, envService, ml, orgID, newApp.ID, "default", "default")
	if err != nil {
		return err
	}

	err = createGitignoredFile(cmd, filepath.Join(appDir, ".env.local"))
	if err != nil {
		return err
	}

	printInitializationSummary(newApp.Name, newApp.AlternateId, newApp.ID, orgID)
	return nil
}
