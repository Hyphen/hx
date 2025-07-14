package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"dario.cat/mergo"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/fsutil"
)

var FS fsutil.FileSystem = fsutil.NewFileSystem()

var (
	ManifestConfigFile = ".hx"
	ManifestSecretFile = ".hxkey"
)

type Config struct {
	ProjectName        *string        `json:"project_name,omitempty"`
	ProjectId          *string        `json:"project_id,omitempty"`
	ProjectAlternateId *string        `json:"project_alternate_id,omitempty"`
	AppName            *string        `json:"app_name,omitempty"`
	AppId              *string        `json:"app_id,omitempty"`
	AppAlternateId     *string        `json:"app_alternate_id,omitempty"`
	OrganizationId     string         `json:"organization_id,omitempty"`
	HyphenAccessToken  *string        `json:"hyphen_access_token,omitempty"`
	HyphenRefreshToken *string        `json:"hyphen_refresh_token,omitempty"`
	HypenIDToken       *string        `json:"hyphen_id_token,omitempty"`
	ExpiryTime         *int64         `json:"expiry_time,omitempty"`
	HyphenAPIKey       *string        `json:"hyphen_api_key,omitempty"`
	IsMonorepo         *bool          `json:"is_monorepo,omitempty"`
	Project            *ConfigProject `json:"project,omitempty"`
	Database           interface{}    `json:"database,omitempty"`
}

func (c *Config) IsMonorepoProject() bool {
	if c.IsMonorepo != nil && *c.IsMonorepo {
		return true
	}
	return false
}

type ConfigProject struct {
	Apps []string `json:"app"`
}

func (w *ConfigProject) AddApp(appDir string) {
	w.Apps = append(w.Apps, appDir)
}

func GlobalInitializeConfig(mc Config) error {
	configDirectory := GetGlobalDirectory()

	manifestConfigFilePath := fmt.Sprintf("%s/%s", configDirectory, ManifestConfigFile)

	return InitializeConfig(mc, manifestConfigFilePath)
}

func GetGlobalDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error retrieving home directory:", err)
		return ""
	}
	return home
}

func UpsertGlobalConfig(mc Config) error {
	globDir := GetGlobalDirectory()

	mc.IsMonorepo = nil //this should always be nil in the global config
	mc.Project = nil    //this should always be nil in the global config

	if err := FS.MkdirAll(globDir, 0755); err != nil {
		return errors.Wrap(err, "Failed to create global directory")
	}

	globManifestFilePath := filepath.Join(globDir, ManifestConfigFile)

	jsonData, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Failed to marshal manifest to JSON")
	}

	// WriteFile will create the file if it doesn't exist, or overwrite it if it does
	if err := FS.WriteFile(globManifestFilePath, jsonData, 0644); err != nil {
		return errors.Wrap(err, "Failed to save manifest")
	}

	return nil
}

func UpsertLocalWorkspace(workspace ConfigProject) error {
	localConfig, err := RestoreLocalConfig()
	if err != nil {
		return errors.Wrap(err, "Failed to restore local config")
	}
	localConfig.Project = &workspace
	jsonData, err := json.MarshalIndent(localConfig, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Failed to marshal manifest to JSON")
	}
	if err := FS.WriteFile(ManifestConfigFile, jsonData, 0644); err != nil {
		return errors.Wrap(err, "Failed to save manifest")
	}
	return nil
}

func AddAppToLocalProject(appDir string) error {
	localConfig, err := RestoreLocalConfig()
	if err != nil {
		return errors.Wrap(err, "Failed to restore local config")
	}
	if localConfig.Project == nil {
		localConfig.Project = &ConfigProject{}
	}
	localConfig.Project.AddApp(appDir)
	jsonData, err := json.MarshalIndent(localConfig, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Failed to marshal manifest to JSON")
	}
	if err := FS.WriteFile(ManifestConfigFile, jsonData, 0644); err != nil {
		return errors.Wrap(err, "Failed to save manifest")
	}
	return nil
}

func InitializeConfig(mc Config, configFile string) error {
	jsonData, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Error encoding JSON")
	}
	err = FS.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", configFile)
	}
	return nil
}

func RestoreConfigFromFile(manifestConfigFile string) (Config, error) {
	var mconfig Config
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), manifestConfigFile)

	globalConfig, err := readAndUnmarshalConfigJSON[Config](globalConfigFile)
	if err == nil {
		mconfig = globalConfig
		hasConfig = true
	} else if !os.IsNotExist(err) {
		return Config{}, err
	}

	localConfig, localConfigErr := readAndUnmarshalConfigJSON[Config](manifestConfigFile)
	if localConfigErr == nil {
		mergeErr := mergo.Merge(&mconfig, localConfig, mergo.WithOverride)
		if mergeErr != nil {
			return Config{}, errors.Wrap(mergeErr, "Error merging your .hx config(s)")
		}
		hasConfig = true
	} else if !os.IsNotExist(localConfigErr) {
		return Config{}, localConfigErr
	}

	if !hasConfig {
		return Config{}, errors.New("No valid .hx found (neither global nor local). Please authenticate using `hx auth` or `hx init`")
	}

	return mconfig, nil
}

func RestoreGlobalConfig() (Config, error) {
	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), ManifestConfigFile)
	return readAndUnmarshalConfigJSON[Config](globalConfigFile)
}

func RestoreLocalConfig() (Config, error) {
	return readAndUnmarshalConfigJSON[Config](ManifestConfigFile)
}

func readAndUnmarshalConfigJSON[T any](filename string) (T, error) {
	var result T

	jsonData, err := FS.ReadFile(filename)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return result, errors.Wrapf(err, "Error decoding JSON file: %s, error: %v", filename, err)
	}

	return result, nil
}

func RestoreConfig() (Config, error) {
	return RestoreConfigFromFile(ManifestConfigFile)
}

func ExistsLocal() bool {
	_, err := FS.Stat(ManifestConfigFile)
	return !os.IsNotExist(err)
}

func ExistsInFolder(folder string) bool {
	configPath := filepath.Join(folder, ManifestConfigFile)
	configExists, err := FS.Stat(configPath)
	if err != nil || configExists == nil {
		return false
	}

	return true
}

func UpsertOrganizationID(organizationID string) error {
	var mconfig Config
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), ManifestConfigFile)
	localConfigFile := ManifestConfigFile
	localConfig, localConfigErr := readAndUnmarshalConfigJSON[Config](localConfigFile)
	if localConfigErr == nil {
		mconfig = localConfig
		hasConfig = true
	}
	if !hasConfig {
		globalConfig, globalConfigErr := readAndUnmarshalConfigJSON[Config](globalConfigFile)
		if globalConfigErr == nil {
			mconfig = globalConfig
			hasConfig = true
		}
	}

	if !hasConfig {
		mc := Config{
			AppName:        nil,
			AppId:          nil,
			AppAlternateId: nil,
			OrganizationId: organizationID,
		}
		jsonData, err := json.MarshalIndent(mc, "", "  ")
		if err != nil {
			return errors.Wrap(err, "Error encoding JSON")
		}
		err = FS.WriteFile(globalConfigFile, jsonData, 0644)
		if err != nil {
			return errors.Wrapf(err, "Error writing file: %s", ManifestConfigFile)
		}
		return nil
	}

	mconfig.OrganizationId = organizationID

	configFile := localConfigFile
	if localConfigErr != nil {
		configFile = globalConfigFile
	}

	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "Error encoding JSON for file: %s", configFile)
	}
	err = FS.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", configFile)
	}

	return nil
}

func UpsertGlobalOrganizationID(organizationID string) error {
	globalDir := GetGlobalDirectory()
	globalConfigFile := filepath.Join(globalDir, ManifestConfigFile)

	var mconfig Config

	// Try to read existing global config
	existingConfig, err := readAndUnmarshalConfigJSON[Config](globalConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "Error reading global config file")
	}

	// If config exists, use it; otherwise, create a new one
	if err == nil {
		mconfig = existingConfig
	}

	// Update or set the OrganizationId
	mconfig.OrganizationId = organizationID

	// Ensure the global directory exists
	if err := FS.MkdirAll(globalDir, 0755); err != nil {
		return errors.Wrap(err, "Failed to create global directory")
	}

	// Marshal the updated config to JSON
	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Error encoding JSON")
	}

	// Write the updated config to the global file
	err = FS.WriteFile(globalConfigFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", globalConfigFile)
	}

	return nil
}

func UpsertProject(project models.Project) error {
	var mconfig Config
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), ManifestConfigFile)
	localConfigFile := ManifestConfigFile

	localConfig, localConfigErr := readAndUnmarshalConfigJSON[Config](localConfigFile)
	if localConfigErr == nil {
		mconfig = localConfig
		hasConfig = true
	}

	if !hasConfig {
		globalConfig, globalConfigErr := readAndUnmarshalConfigJSON[Config](globalConfigFile)
		if globalConfigErr == nil {
			mconfig = globalConfig
			hasConfig = true
		}
	}

	if !hasConfig {
		mc := Config{
			ProjectId:          project.ID,
			ProjectName:        &project.Name,
			ProjectAlternateId: &project.AlternateID,
			AppName:            nil,
			AppId:              nil,
			AppAlternateId:     nil,
			OrganizationId:     "",
		}
		jsonData, err := json.MarshalIndent(mc, "", "  ")
		if err != nil {
			return errors.Wrap(err, "Error encoding JSON")
		}
		err = FS.WriteFile(globalConfigFile, jsonData, 0644)
		if err != nil {
			return errors.Wrapf(err, "Error writing file: %s", ManifestConfigFile)
		}
		return nil
	}

	mconfig.ProjectId = project.ID
	mconfig.ProjectName = &project.Name
	mconfig.ProjectAlternateId = &project.AlternateID

	configFile := localConfigFile
	if localConfigErr != nil {
		configFile = globalConfigFile
	}

	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "Error encoding JSON for file: %s", configFile)
	}
	err = FS.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", configFile)
	}

	return nil
}

func UpsertGlobalProject(project models.Project) error {
	globalDir := GetGlobalDirectory()
	globalConfigFile := filepath.Join(globalDir, ManifestConfigFile)

	var mconfig Config

	// Try to read existing global config
	existingConfig, err := readAndUnmarshalConfigJSON[Config](globalConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "Error reading global config file")
	}

	// If config exists, use it; otherwise, create a new one
	if err == nil {
		mconfig = existingConfig
	}

	// Update or set the project fields
	mconfig.ProjectId = project.ID
	mconfig.ProjectName = &project.Name
	mconfig.ProjectAlternateId = &project.AlternateID

	// Ensure the global directory exists
	if err := FS.MkdirAll(globalDir, 0755); err != nil {
		return errors.Wrap(err, "Failed to create global directory")
	}

	// Marshal the updated config to JSON
	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Error encoding JSON")
	}

	// Write the updated config to the global file
	err = FS.WriteFile(globalConfigFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", globalConfigFile)
	}

	return nil
}
