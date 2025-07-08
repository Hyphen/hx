package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/cmd/initapp"
	"github.com/Hyphen/cli/cmd/initproject"
	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/database"
	"github.com/Hyphen/cli/internal/env"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/secret"
	"github.com/Hyphen/cli/internal/secretkey"
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

	c, err := config.RestoreConfig()
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	s, err := secret.LoadSecret(c.OrganizationId, *c.ProjectId, true)
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

		cleanPath := filepath.Clean(appPath)

		if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
			printer.Error(cmd, fmt.Errorf("Directory does not exist: %s", cleanPath))
			continue
		}

		if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
			printer.Error(cmd, fmt.Errorf("directory does not exist: %s", cleanPath))
			continue
		}

		currentDir, err := os.Getwd()
		if err != nil {
			printer.Error(cmd, fmt.Errorf("Failed to get current directory: %w", err))
			return
		}

		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("Invalid path: %w", err))
			continue
		}

		// Check if path is within current directory using filepath.Rel
		relPath, err := filepath.Rel(currentDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			printer.Error(cmd, fmt.Errorf("Invalid path: must be within current directory"))
			continue
		}

		printer.Info(fmt.Sprintf("Initializing app %s", cleanPath))
		err = initializeMonorepoApp(cmd, cleanPath, orgID, c, appService, envService, s)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("Failed to initialize app %s: %w", cleanPath, err))
			continue
		}

		if err := config.AddAppToLocalProject(cleanPath); err != nil {
			printer.Error(cmd, fmt.Errorf("Failed to add app to local project: %w", err))
			continue
		}

		moreApps := prompt.PromptYesNo(cmd, "Do you have another app?", false)
		if !moreApps.Confirmed {
			break
		}
	}
}

func initializeMonorepoApp(cmd *cobra.Command, appDir string, orgID string, mc config.Config, appService *app.AppService, envService *env.EnvService, monorepoSecret models.Secret) error {
	// Get the app name from directory and prompt for confirmation/new name
	defaultAppName := filepath.Base(appDir)
	response := prompt.PromptYesNo(cmd, fmt.Sprintf("Use the directory name '%s' as the app name?", defaultAppName), true)

	var appName string
	if !response.Confirmed {
		if response.IsFlag {
			return fmt.Errorf("Operation cancelled due to --no flag for app: %s", defaultAppName)
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
				return fmt.Errorf("Operation cancelled due to --no flag for app ID: %s", defaultAppAlternateId)
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
			return fmt.Errorf("Operation cancelled for app: %s", appName)
		}

		newApp = *existingApp
	}

	mcl := config.Config{
		ProjectId:          mc.ProjectId,
		ProjectAlternateId: mc.ProjectAlternateId,
		ProjectName:        mc.ProjectName,
		OrganizationId:     mc.OrganizationId,
		AppName:            &newApp.Name,
		AppAlternateId:     &newApp.AlternateId,
		AppId:              &newApp.ID,
	}

	// Initialize only the config file in the app directory
	err = config.InitializeConfig(mcl, filepath.Join(appDir, config.ManifestConfigFile))
	if err != nil {
		return err
	}

	// List environments for the project
	environments, err := envService.ListEnvironments(orgID, *mcl.ProjectId, 100, 1)
	if err != nil {
		return err
	}

	for _, e := range environments {
		envName := strings.ToLower(e.Name)
		envID := e.ID
		err = CreateAndPushEmptyEnvFileMonorepo(cmd, envService, mcl, monorepoSecret, orgID, newApp.ID, envID, envName, appDir)
		if err != nil {
			return err
		}
	}

	err = CreateAndPushEmptyEnvFileMonorepo(cmd, envService, mcl, monorepoSecret, orgID, newApp.ID, "default", "default", appDir)
	if err != nil {
		return err
	}

	err = CreateGitignoredFileMonorepo(cmd, filepath.Join(appDir, ".env.local"), ".env.local")
	if err != nil {
		return err
	}

	initapp.PrintInitializationSummary(newApp.Name, newApp.AlternateId, newApp.ID, orgID, *mc.ProjectAlternateId)
	return nil
}

func CreateAndPushEmptyEnvFileMonorepo(cmd *cobra.Command, envService *env.EnvService, c config.Config, s models.Secret, orgID, appID, envID, envName, appDir string) error {
	envFileName, err := env.GetFileName(envName)
	if err != nil {
		return err
	}

	// Use filepath.Join to create the full path including appDir
	fullEnvPath := filepath.Join(appDir, envFileName)

	// Ensure the directory exists
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("Failed to create directory %s: %w", appDir, err)
	}

	err = CreateGitignoredFileMonorepo(cmd, fullEnvPath, envFileName)
	if err != nil {
		return err
	}

	// Build an Env struct from that new empty file
	envStruct, err := env.GetLocalEncryptedEnv(envName, &appDir, s)
	if err != nil {
		return err
	}

	version := 1
	envStruct.Version = &version

	if err := envService.PutEnvironmentEnv(orgID, appID, envID, s.SecretKeyId, envStruct); err != nil {
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

	newEnvDecrypted, err := envStruct.DecryptData(secretkey.FromBase64(s.Base64SecretKey))
	if err != nil {
		return err
	}

	secretKey := database.SecretKey{
		ProjectId: *c.ProjectId,
		AppId:     *c.AppId,
		EnvName:   envName,
	}

	if err := db.UpsertSecret(secretKey, newEnvDecrypted, version); err != nil {
		return fmt.Errorf("Failed to save local environment: %w", err)
	}

	return nil
}

func CreateGitignoredFileMonorepo(cmd *cobra.Command, fullPath, fileName string) error {
	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Failed to create directory %s: %w", dir, err)
	}

	// check if the file already exists.
	if _, err := os.Stat(fullPath); err == nil {
		// do not recreate. file exists already.
		return nil
	}

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		printer.Error(cmd, fmt.Errorf("Error creating %s: %w", fullPath, err))
		return err
	}
	defer file.Close()

	// Write '# KEY=Value' to the file
	_, err = file.WriteString("# Example\n# KEY=Value\n")
	if err != nil {
		printer.Error(cmd, fmt.Errorf("Error writing to %s: %w", fullPath, err))
		return err
	}

	if err := gitutil.EnsureGitignore(fileName); err != nil {
		printer.Error(cmd, fmt.Errorf("Error adding %s to .gitignore: %w. Please do this manually if you wish", fileName, err))
		// don't error here. Keep going.
	}

	return nil
}
