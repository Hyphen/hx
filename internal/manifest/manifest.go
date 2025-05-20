package manifest

// import (
// 	"encoding/json"
// 	"fmt"
// 	"os"
// 	"path/filepath"

// 	"dario.cat/mergo"
// 	"github.com/Hyphen/cli/internal/config"
// 	"github.com/Hyphen/cli/internal/key"
// 	"github.com/Hyphen/cli/internal/secretkey"
// 	"github.com/Hyphen/cli/pkg/errors"
// 	"github.com/Hyphen/cli/pkg/fsutil"
// )

// var FS fsutil.FileSystem = fsutil.NewFileSystem()

// type Manifest struct {
// 	config.Config
// 	key.Secret
// }

// // It will detect if its an app of a monorepo, and will not create the .hxkey
// func LocalInitialize(mc config.Config) (Manifest, error) {
// 	err := config.InitializeConfig(mc)
// 	if err != nil {
// 		return Manifest{}, errors.Wrap(err, "Failed to initialize manifest config")
// 	}

// 	ms, err := key.LoadSecret()
// 	if err != nil {
// 		return Manifest{}, fmt.Errorf("failed to load manifest secret: %w", err)
// 	}

// 	m := Manifest{
// 		mc,
// 		ms,
// 	}

// 	return m, nil
// }

// func GetGlobalDirectory() string {
// 	home, err := os.UserHomeDir()
// 	if err != nil {
// 		fmt.Println("Error retrieving home directory:", err)
// 		return ""
// 	}
// 	return home
// }

// func UpsertGlobalConfig(mc Config) error {
// 	globDir := GetGlobalDirectory()

// 	mc.IsMonorepo = nil //this should always be nil in the global config
// 	mc.Project = nil    //this should always be nil in the global config

// 	if err := FS.MkdirAll(globDir, 0755); err != nil {
// 		return errors.Wrap(err, "Failed to create global directory")
// 	}

// 	globManifestFilePath := filepath.Join(globDir, ManifestConfigFile)

// 	jsonData, err := json.MarshalIndent(mc, "", "  ")
// 	if err != nil {
// 		return errors.Wrap(err, "Failed to marshal manifest to JSON")
// 	}

// 	// WriteFile will create the file if it doesn't exist, or overwrite it if it does
// 	if err := FS.WriteFile(globManifestFilePath, jsonData, 0644); err != nil {
// 		return errors.Wrap(err, "Failed to save manifest")
// 	}

// 	return nil
// }

// func UpsertLocalWorkspace(workspace Project) error {
// 	localConfig, err := RestoreLocalConfig()
// 	if err != nil {
// 		return errors.Wrap(err, "Failed to restore local config")
// 	}
// 	localConfig.Project = &workspace
// 	jsonData, err := json.MarshalIndent(localConfig, "", "  ")
// 	if err != nil {
// 		return errors.Wrap(err, "Failed to marshal manifest to JSON")
// 	}
// 	if err := FS.WriteFile(ManifestConfigFile, jsonData, 0644); err != nil {
// 		return errors.Wrap(err, "Failed to save manifest")
// 	}
// 	return nil
// }

// func AddAppToLocalProject(appDir string) error {
// 	localConfig, err := RestoreLocalConfig()
// 	if err != nil {
// 		return errors.Wrap(err, "Failed to restore local config")
// 	}
// 	if localConfig.Project == nil {
// 		localConfig.Project = &Project{}
// 	}
// 	localConfig.Project.AddApp(appDir)
// 	jsonData, err := json.MarshalIndent(localConfig, "", "  ")
// 	if err != nil {
// 		return errors.Wrap(err, "Failed to marshal manifest to JSON")
// 	}
// 	if err := FS.WriteFile(ManifestConfigFile, jsonData, 0644); err != nil {
// 		return errors.Wrap(err, "Failed to save manifest")
// 	}
// 	return nil
// }

// func InitializeConfig(mc Config, configFile string) error {
// 	jsonData, err := json.MarshalIndent(mc, "", "  ")
// 	if err != nil {
// 		return errors.Wrap(err, "Error encoding JSON")
// 	}
// 	err = FS.WriteFile(configFile, jsonData, 0644)
// 	if err != nil {
// 		return errors.Wrapf(err, "Error writing file: %s", configFile)
// 	}
// 	return nil
// }

// func InitializeSecret(secretFile string) (Secret, error) {
// 	useVinz := true

// 	sk, err := secretkey.New()
// 	if err != nil {
// 		return Secret{}, errors.Wrap(err, "Failed to create new secret key")
// 	}

// 	ms := NewSecret(sk)

// 	jsonData, err := json.MarshalIndent(ms, "", "  ")
// 	if err != nil {
// 		return Secret{}, errors.Wrap(err, "Error encoding JSON")
// 	}

// 	if useVinz {

// 	} else {
// 		err = FS.WriteFile(secretFile, jsonData, 0644)
// 		if err != nil {
// 			return Secret{}, errors.Wrapf(err, "Error writing file: %s", secretFile)
// 		}
// 	}

// 	return ms, nil
// }

// func Initialize(mc Config, secretFile, configFile string) (Manifest, error) {
// 	err := InitializeConfig(mc, configFile)
// 	if err != nil {
// 		return Manifest{}, errors.Wrap(err, "Failed to initialize manifest config")
// 	}

// 	ms, err := InitializeSecret(secretFile)
// 	if err != nil {
// 		return Manifest{}, errors.Wrap(err, "Failed to initialize manifest secret")
// 	}

// 	m := Manifest{
// 		mc,
// 		ms,
// 	}

// 	return m, nil
// }

// func RestoreConfigFromFile(manifestConfigFile string) (Config, error) {
// 	var mconfig Config
// 	var hasConfig bool

// 	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), manifestConfigFile)

// 	globalConfig, err := readAndUnmarshalConfigJSON[Config](globalConfigFile)
// 	if err == nil {
// 		mconfig = globalConfig
// 		hasConfig = true
// 	} else if !os.IsNotExist(err) {
// 		return Config{}, err
// 	}

// 	localConfig, localConfigErr := readAndUnmarshalConfigJSON[Config](manifestConfigFile)
// 	if localConfigErr == nil {
// 		mergeErr := mergo.Merge(&mconfig, localConfig, mergo.WithOverride)
// 		if mergeErr != nil {
// 			return Config{}, errors.Wrap(mergeErr, "Error merging your .hx config(s)")
// 		}
// 		hasConfig = true
// 	} else if !os.IsNotExist(localConfigErr) {
// 		return Config{}, localConfigErr
// 	}

// 	if !hasConfig {
// 		return Config{}, errors.New("No valid .hx found (neither global nor local). Please authenticate using `hx auth` or `hx init`")
// 	}

// 	return mconfig, nil
// }

// func RestoreSecretFromMonorepo() (Secret, error) {
// 	// Start from current directory
// 	currentDir, err := os.Getwd()
// 	if err != nil {
// 		return Secret{}, errors.Wrap(err, "Failed to get current working directory")
// 	}

// 	// Keep traversing up until we find a monorepo config or hit root
// 	for {
// 		// Check for .hx file in current directory
// 		configPath := filepath.Join(currentDir, ManifestConfigFile)
// 		config, err := readAndUnmarshalConfigJSON[Config](configPath)

// 		// If we can read the config and it's a monorepo
// 		if err == nil && config.IsMonorepoProject() {
// 			// Look for .hxkey in the same directory
// 			secretPath := filepath.Join(currentDir, ManifestSecretFile)
// 			secret, err := readAndUnmarshalConfigJSON[Secret](secretPath)
// 			if err == nil {
// 				return secret, nil
// 			}
// 			return Secret{}, errors.Wrapf(err, "Found monorepo config at %s but failed to read secret file", currentDir)
// 		}

// 		// Move up one directory
// 		parentDir := filepath.Dir(currentDir)

// 		// Check if we've hit the root directory
// 		if parentDir == currentDir {
// 			return Secret{}, errors.New("No monorepo configuration found in parent directories")
// 		}

// 		currentDir = parentDir
// 	}
// }

// func RestoreSecretFromFile(manifestSecretFile string) (Secret, error) {
// 	monorepoSecret, err := RestoreSecretFromMonorepo()
// 	if err == nil {
// 		return monorepoSecret, nil
// 	}

// 	var secret Secret
// 	//var hasSecret bool

// 	globalSecretFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), manifestSecretFile)

// 	globalSecret, err := readAndUnmarshalConfigJSON[Secret](globalSecretFile)
// 	if err == nil {
// 		secret = globalSecret
// 		//hasSecret = true
// 	} else if !os.IsNotExist(err) {
// 		return Secret{}, err
// 	}

// 	localSecret, localSecretErr := readAndUnmarshalConfigJSON[Secret](manifestSecretFile)
// 	if localSecretErr == nil {
// 		mergeErr := mergo.Merge(&secret, localSecret, mergo.WithOverride)
// 		if mergeErr != nil {
// 			return Secret{}, errors.Wrap(mergeErr, "Error merging your .hxkey secret(s)")
// 		}
// 		//hasSecret = true
// 	} else if !os.IsNotExist(localSecretErr) {
// 		return Secret{}, localSecretErr
// 	}

// 	// if !hasSecret {
// 	// 	return Secret{}, errors.New("No valid .hxkey found (neither global, local, nor monorepo). Please init an app using `hx init`")
// 	// }

// 	return secret, nil
// }

// func RestoreFromFile(manifestConfigFile, manifestSecretFile string) (Manifest, error) {
// 	mconfig, configErr := RestoreConfigFromFile(manifestConfigFile)
// 	if configErr != nil {
// 		return Manifest{}, configErr
// 	}

// 	secret, secretErr := RestoreSecretFromFile(manifestSecretFile)
// 	if secretErr != nil {
// 		return Manifest{}, secretErr
// 	}

// 	return Manifest{
// 		mconfig,
// 		secret,
// 	}, nil
// }

// func RestoreGlobalConfig() (Config, error) {
// 	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), ManifestConfigFile)
// 	return readAndUnmarshalConfigJSON[Config](globalConfigFile)
// }

// func RestoreLocalConfig() (Config, error) {
// 	return readAndUnmarshalConfigJSON[Config](ManifestConfigFile)
// }

// func readAndUnmarshalConfigJSON[T any](filename string) (T, error) {
// 	var result T

// 	jsonData, err := FS.ReadFile(filename)
// 	if err != nil {
// 		return result, err
// 	}

// 	err = json.Unmarshal(jsonData, &result)
// 	if err != nil {
// 		return result, errors.Wrapf(err, "Error decoding JSON file: %s, error: %v", filename, err)
// 	}

// 	return result, nil
// }

// func Restore() (Manifest, error) {
// 	return RestoreFromFile(ManifestConfigFile, ManifestSecretFile)
// }

// func RestoreConfig() (Config, error) {
// 	return RestoreConfigFromFile(ManifestConfigFile)
// }

// func ExistsLocal() bool {
// 	_, err := FS.Stat(ManifestConfigFile)
// 	return !os.IsNotExist(err)
// }

// func ExistsInFolder(folder string) bool {
// 	configPath := filepath.Join(folder, ManifestConfigFile)
// 	configExists, err := FS.Stat(configPath)
// 	if err != nil || configExists == nil {
// 		return false
// 	}

// 	return true
// }

// func UpsertOrganizationID(organizationID string) error {
// 	var mconfig Config
// 	var hasConfig bool

// 	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), ManifestConfigFile)
// 	localConfigFile := ManifestConfigFile
// 	localConfig, localConfigErr := readAndUnmarshalConfigJSON[Config](localConfigFile)
// 	if localConfigErr == nil {
// 		mconfig = localConfig
// 		hasConfig = true
// 	}
// 	if !hasConfig {
// 		globalConfig, globalConfigErr := readAndUnmarshalConfigJSON[Config](globalConfigFile)
// 		if globalConfigErr == nil {
// 			mconfig = globalConfig
// 			hasConfig = true
// 		}
// 	}

// 	if !hasConfig {
// 		mc := Config{
// 			AppName:        nil,
// 			AppId:          nil,
// 			AppAlternateId: nil,
// 			OrganizationId: organizationID,
// 		}
// 		jsonData, err := json.MarshalIndent(mc, "", "  ")
// 		if err != nil {
// 			return errors.Wrap(err, "Error encoding JSON")
// 		}
// 		err = FS.WriteFile(globalConfigFile, jsonData, 0644)
// 		if err != nil {
// 			return errors.Wrapf(err, "Error writing file: %s", ManifestConfigFile)
// 		}
// 		return nil
// 	}

// 	mconfig.OrganizationId = organizationID

// 	configFile := localConfigFile
// 	if localConfigErr != nil {
// 		configFile = globalConfigFile
// 	}

// 	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
// 	if err != nil {
// 		return errors.Wrapf(err, "Error encoding JSON for file: %s", configFile)
// 	}
// 	err = FS.WriteFile(configFile, jsonData, 0644)
// 	if err != nil {
// 		return errors.Wrapf(err, "Error writing file: %s", configFile)
// 	}

// 	return nil
// }

// func UpsertGlobalOrganizationID(organizationID string) error {
// 	globalDir := GetGlobalDirectory()
// 	globalConfigFile := filepath.Join(globalDir, ManifestConfigFile)

// 	var mconfig Config

// 	// Try to read existing global config
// 	existingConfig, err := readAndUnmarshalConfigJSON[Config](globalConfigFile)
// 	if err != nil && !os.IsNotExist(err) {
// 		return errors.Wrap(err, "Error reading global config file")
// 	}

// 	// If config exists, use it; otherwise, create a new one
// 	if err == nil {
// 		mconfig = existingConfig
// 	}

// 	// Update or set the OrganizationId
// 	mconfig.OrganizationId = organizationID

// 	// Ensure the global directory exists
// 	if err := FS.MkdirAll(globalDir, 0755); err != nil {
// 		return errors.Wrap(err, "Failed to create global directory")
// 	}

// 	// Marshal the updated config to JSON
// 	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
// 	if err != nil {
// 		return errors.Wrap(err, "Error encoding JSON")
// 	}

// 	// Write the updated config to the global file
// 	err = FS.WriteFile(globalConfigFile, jsonData, 0644)
// 	if err != nil {
// 		return errors.Wrapf(err, "Error writing file: %s", globalConfigFile)
// 	}

// 	return nil
// }

// func UpsertProjectID(projectID string) error {
// 	var mconfig Config
// 	var hasConfig bool

// 	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), ManifestConfigFile)
// 	localConfigFile := ManifestConfigFile

// 	localConfig, localConfigErr := readAndUnmarshalConfigJSON[Config](localConfigFile)
// 	if localConfigErr == nil {
// 		mconfig = localConfig
// 		hasConfig = true
// 	}

// 	if !hasConfig {
// 		globalConfig, globalConfigErr := readAndUnmarshalConfigJSON[Config](globalConfigFile)
// 		if globalConfigErr == nil {
// 			mconfig = globalConfig
// 			hasConfig = true
// 		}
// 	}

// 	if !hasConfig {
// 		mc := Config{
// 			ProjectId:      &projectID,
// 			AppName:        nil,
// 			AppId:          nil,
// 			AppAlternateId: nil,
// 			OrganizationId: "",
// 		}
// 		jsonData, err := json.MarshalIndent(mc, "", "  ")
// 		if err != nil {
// 			return errors.Wrap(err, "Error encoding JSON")
// 		}
// 		err = FS.WriteFile(globalConfigFile, jsonData, 0644)
// 		if err != nil {
// 			return errors.Wrapf(err, "Error writing file: %s", ManifestConfigFile)
// 		}
// 		return nil
// 	}

// 	mconfig.ProjectId = &projectID

// 	configFile := localConfigFile
// 	if localConfigErr != nil {
// 		configFile = globalConfigFile
// 	}

// 	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
// 	if err != nil {
// 		return errors.Wrapf(err, "Error encoding JSON for file: %s", configFile)
// 	}
// 	err = FS.WriteFile(configFile, jsonData, 0644)
// 	if err != nil {
// 		return errors.Wrapf(err, "Error writing file: %s", configFile)
// 	}

// 	return nil
// }

// func UpsertGlobalProjectID(projectID string) error {
// 	globalDir := GetGlobalDirectory()
// 	globalConfigFile := filepath.Join(globalDir, ManifestConfigFile)

// 	var mconfig Config

// 	// Try to read existing global config
// 	existingConfig, err := readAndUnmarshalConfigJSON[Config](globalConfigFile)
// 	if err != nil && !os.IsNotExist(err) {
// 		return errors.Wrap(err, "Error reading global config file")
// 	}

// 	// If config exists, use it; otherwise, create a new one
// 	if err == nil {
// 		mconfig = existingConfig
// 	}

// 	// Update or set the ProjectId
// 	mconfig.ProjectId = &projectID

// 	// Ensure the global directory exists
// 	if err := FS.MkdirAll(globalDir, 0755); err != nil {
// 		return errors.Wrap(err, "Failed to create global directory")
// 	}

// 	// Marshal the updated config to JSON
// 	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
// 	if err != nil {
// 		return errors.Wrap(err, "Error encoding JSON")
// 	}

// 	// Write the updated config to the global file
// 	err = FS.WriteFile(globalConfigFile, jsonData, 0644)
// 	if err != nil {
// 		return errors.Wrapf(err, "Error writing file: %s", globalConfigFile)
// 	}

// 	return nil
// }

// func UpsertLocalSecret(secret Secret) error {
// 	localSecretFile := ManifestSecretFile

// 	jsonData, err := json.MarshalIndent(secret, "", "  ")
// 	if err != nil {
// 		return errors.Wrap(err, "Error encoding JSON")
// 	}

// 	err = FS.WriteFile(localSecretFile, jsonData, 0644)
// 	if err != nil {
// 		return errors.Wrapf(err, "Error writing file: %s", localSecretFile)
// 	}

// 	return nil
// }
