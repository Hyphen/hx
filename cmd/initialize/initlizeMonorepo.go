package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/cmd/initapp"
	"github.com/Hyphen/cli/cmd/initproject"
	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
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

	initproject.IsMonorepo = true
	initproject.RunInitProject(cmd, args)

	m, err := manifest.Restore()
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	for {
		appPath, err := prompt.PromptString(cmd, "Provide the path to an application:")
		if err != nil {
			printer.Error(cmd, err)
			return
		}

		printer.Info(fmt.Sprintf("Initializing app %s", appPath))
		err = initializeMonorepoApp(cmd, appPath, orgID, m.Config, appService, envService, m.Secret)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to initialize app %s: %w", appPath, err))
			continue
		}

		moreApps := prompt.PromptYesNo(cmd, "Do you have another?", false)
		if !moreApps.Confirmed {
			break
		}
	}
}

func initializeMonorepoApp(cmd *cobra.Command, appDir string, orgID string, mc manifest.Config, appService *app.AppService, envService *env.EnvService, monorepoSecret manifest.Secret) error {
	// Get the app name from directory and prompt for confirmation/new name
	defaultAppName := filepath.Base(appDir)
	response := prompt.PromptYesNo(cmd, fmt.Sprintf("Use the directory name '%s' as the app name?", defaultAppName), true)

	var appName string
	if !response.Confirmed {
		if response.IsFlag {
			return fmt.Errorf("operation cancelled due to --no flag for app: %s", defaultAppName)
		}

		// Prompt for a new app name
		promptedName, err := prompt.PromptString(cmd, "What would you like the app name to be?")
		if err != nil {
			return err
		}
		appName = promptedName
	} else {
		appName = defaultAppName
	}

	// Generate and prompt for app ID
	defaultAppAlternateId := initapp.GenerateDefaultAppId(appName)
	err := app.CheckAppId(defaultAppAlternateId)
	var appAlternateId string

	if err != nil {
		suggestedID := strings.TrimSpace(strings.Split(err.Error(), ":")[1])
		response := prompt.PromptYesNo(cmd, fmt.Sprintf("Invalid app ID. Do you want to use the suggested ID [%s]?", suggestedID), true)

		if !response.Confirmed {
			if response.IsFlag {
				return fmt.Errorf("operation cancelled due to --no flag for app ID: %s", defaultAppAlternateId)
			}

			// Prompt for a custom app ID
			for {
				promptedID, err := prompt.PromptString(cmd, "Enter a custom app ID:")
				if err != nil {
					return err
				}

				err = app.CheckAppId(promptedID)
				if err == nil {
					appAlternateId = promptedID
					break
				}
				printer.Warning("Invalid app ID. Please try again.")
			}
		} else {
			appAlternateId = suggestedID
		}
	} else {
		appAlternateId = defaultAppAlternateId
	}

	// Create the app in Hyphen
	newApp, err := appService.CreateApp(orgID, *mc.ProjectId, appAlternateId, appName)
	if err != nil {
		if !errors.Is(err, errors.ErrConflict) {
			return err
		}

		existingApp, handleErr := initapp.HandleExistingApp(cmd, *appService, orgID, appAlternateId)
		if handleErr != nil {
			return handleErr
		}
		if existingApp == nil {
			return fmt.Errorf("operation cancelled for app: %s", appName)
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

	for _, e := range environments {
		envName := strings.ToLower(e.Name)
		envID := e.ID
		err = CreateAndPushEmptyEnvFileMonorepo(cmd, envService, ml, orgID, newApp.ID, envID, envName, appDir)
		if err != nil {
			return err
		}
	}

	err = CreateAndPushEmptyEnvFileMonorepo(cmd, envService, ml, orgID, newApp.ID, "default", "default", appDir)
	if err != nil {
		return err
	}

	err = CreateGitignoredFileMonorepo(cmd, filepath.Join(appDir, ".env.local"), ".env.local")
	if err != nil {
		return err
	}

	initapp.PrintInitializationSummary(newApp.Name, newApp.AlternateId, newApp.ID, orgID)
	return nil
}

func CreateAndPushEmptyEnvFileMonorepo(cmd *cobra.Command, envService *env.EnvService, m manifest.Manifest, orgID, appID, envID, envName, appDir string) error {
	envFileName, err := env.GetFileName(envName)
	if err != nil {
		return err
	}

	// Use filepath.Join to create the full path including appDir
	fullEnvPath := filepath.Join(appDir, envFileName)

	// Ensure the directory exists
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", appDir, err)
	}

	err = CreateGitignoredFileMonorepo(cmd, fullEnvPath, envFileName)
	if err != nil {
		return err
	}

	// Build an Env struct from that new empty file
	envStruct, err := env.GetLocalEncryptedEnv(envName, &appDir, m)
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

	newEnvDecrypted, err := envStruct.DecryptData(m.GetSecretKey())
	if err != nil {
		return err
	}

	secretKey := database.SecretKey{
		ProjectId: *m.ProjectId,
		AppId:     *m.AppId,
		EnvName:   envName,
	}

	if err := db.UpsertSecret(secretKey, newEnvDecrypted, version); err != nil {
		return fmt.Errorf("failed to save local environment: %w", err)
	}

	return nil
}

func CreateGitignoredFileMonorepo(cmd *cobra.Command, fullPath, fileName string) error {
	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// check if the file already exists.
	if _, err := os.Stat(fullPath); err == nil {
		// do not recreate. file exists already.
		return nil
	}

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		printer.Error(cmd, fmt.Errorf("error creating %s: %w", fullPath, err))
		return err
	}
	defer file.Close()

	// Write '# KEY=Value' to the file
	_, err = file.WriteString("# Example\n# KEY=Value\n")
	if err != nil {
		printer.Error(cmd, fmt.Errorf("error writing to %s: %w", fullPath, err))
		return err
	}

	if err := gitutil.EnsureGitignore(fileName); err != nil {
		printer.Error(cmd, fmt.Errorf("error adding %s to .gitignore: %w. Please do this manually if you wish", fileName, err))
		// don't error here. Keep going.
	}

	return nil
}
