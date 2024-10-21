package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dario.cat/mergo"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/fsutil"
)

var FS fsutil.FileSystem = fsutil.NewFileSystem()

var (
	ManifestConfigFile = ".hx"
	ManifestSecretFile = ".hxkey"
)

type ManifestConfig struct {
	ProjectName        *string     `json:"project_name,omitempty"`
	ProjectId          *string     `json:"project_id,omitempty"`
	ProjectAlternateId *string     `json:"project_alternate_id,omitempty"`
	AppName            *string     `json:"app_name,omitempty"`
	AppId              *string     `json:"app_id,omitempty"`
	AppAlternateId     *string     `json:"app_alternate_id,omitempty"`
	OrganizationId     string      `json:"organization_id,omitempty"`
	HyphenAccessToken  *string     `json:"hyphen_access_token,omitempty"`
	HyphenRefreshToken *string     `json:"hyphen_refresh_token,omitempty"`
	HypenIDToken       *string     `json:"hyphen_id_token,omitempty"`
	ExpiryTime         *int64      `json:"expiry_time,omitempty"`
	HyphenAPIKey       *string     `json:"hyphen_api_key,omitempty"`
	Database           interface{} `json:"database,omitempty"`
}

type Manifest struct {
	ManifestConfig
	ManifestSecret
}

type ManifestSecret struct {
	SecretKeyId int64  `json:"secret_key_id"`
	SecretKey   string `json:"secret_key"`
}

func NewSecret(sk *secretkey.SecretKey) ManifestSecret {
	return ManifestSecret{
		SecretKeyId: time.Now().Unix(),
		SecretKey:   sk.Base64(),
	}
}

func (m *Manifest) GetSecretKey() *secretkey.SecretKey {
	return secretkey.FromBase64(m.SecretKey)
}

func LocalInitialize(mc ManifestConfig) (Manifest, error) {
	return Initialize(mc, ManifestSecretFile, ManifestConfigFile)
}

func GlobalInitializeManifestConfig(mc ManifestConfig) error {
	configDirectory := GetGlobalDirectory()

	manifestConfigFilePath := fmt.Sprintf("%s/%s", configDirectory, ManifestConfigFile)

	return InitializeManifestConfig(mc, manifestConfigFilePath)
}

func GetGlobalDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error retrieving home directory:", err)
		return ""
	}
	return home
}

func UpsertGlobalManifestConfig(mc ManifestConfig) error {
	globDir := GetGlobalDirectory()

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

func InitializeManifestConfig(mc ManifestConfig, configFile string) error {
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

func InitializeManifestSecret(secretFile string) (ManifestSecret, error) {
	sk, err := secretkey.New()
	if err != nil {
		return ManifestSecret{}, errors.Wrap(err, "Failed to create new secret key")
	}

	ms := NewSecret(sk)

	jsonData, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return ManifestSecret{}, errors.Wrap(err, "Error encoding JSON")
	}
	err = FS.WriteFile(secretFile, jsonData, 0644)
	if err != nil {
		return ManifestSecret{}, errors.Wrapf(err, "Error writing file: %s", secretFile)
	}

	return ms, nil
}

func Initialize(mc ManifestConfig, secretFile, configFile string) (Manifest, error) {
	err := InitializeManifestConfig(mc, configFile)
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Failed to initialize manifest config")
	}

	ms, err := InitializeManifestSecret(secretFile)
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Failed to initialize manifest secret")
	}

	m := Manifest{
		mc,
		ms,
	}

	return m, nil
}

func RestoreManifestConfigFromFile(manifestConfigFile string) (ManifestConfig, error) {
	var mconfig ManifestConfig
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), manifestConfigFile)

	globalConfig, err := readAndUnmarshalConfigJSON[ManifestConfig](globalConfigFile)
	if err == nil {
		mconfig = globalConfig
		hasConfig = true
	} else if !os.IsNotExist(err) {
		return ManifestConfig{}, err
	}

	localConfig, localConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](manifestConfigFile)
	if localConfigErr == nil {
		mergeErr := mergo.Merge(&mconfig, localConfig, mergo.WithOverride)
		if mergeErr != nil {
			return ManifestConfig{}, errors.Wrap(mergeErr, "Error merging your .hx config(s)")
		}
		hasConfig = true
	} else if !os.IsNotExist(localConfigErr) {
		return ManifestConfig{}, localConfigErr
	}

	if !hasConfig {
		return ManifestConfig{}, errors.New("No valid .hx found (neither global nor local). Please authenticate using `hx auth` or `hx init`")
	}

	return mconfig, nil
}

func RestoreManifestSecretFromFile(manifestSecretFile string) (ManifestSecret, error) {
	var secret ManifestSecret
	var hasSecret bool

	globalSecretFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), manifestSecretFile)

	globalSecret, err := readAndUnmarshalConfigJSON[ManifestSecret](globalSecretFile)
	if err == nil {
		secret = globalSecret
		hasSecret = true
	} else if !os.IsNotExist(err) {
		return ManifestSecret{}, err
	}

	localSecret, localSecretErr := readAndUnmarshalConfigJSON[ManifestSecret](manifestSecretFile)
	if localSecretErr == nil {
		mergeErr := mergo.Merge(&secret, localSecret, mergo.WithOverride)
		if mergeErr != nil {
			return ManifestSecret{}, errors.Wrap(mergeErr, "Error merging your .hxkey secret(s)")
		}
		hasSecret = true
	} else if !os.IsNotExist(localSecretErr) {
		return ManifestSecret{}, localSecretErr
	}

	if !hasSecret {
		return ManifestSecret{}, errors.New("No valid .hxkey found (neither global nor local). Please authenticate using `hx init`")
	}

	return secret, nil
}

func RestoreFromFile(manifestConfigFile, manifestSecretFile string) (Manifest, error) {
	mconfig, configErr := RestoreManifestConfigFromFile(manifestConfigFile)
	if configErr != nil {
		return Manifest{}, configErr
	}

	secret, secretErr := RestoreManifestSecretFromFile(manifestSecretFile)
	if secretErr != nil {
		return Manifest{}, secretErr
	}

	return Manifest{
		mconfig,
		secret,
	}, nil
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

func Restore() (Manifest, error) {
	return RestoreFromFile(ManifestConfigFile, ManifestSecretFile)
}

func RestoreManifestConfig() (ManifestConfig, error) {
	return RestoreManifestConfigFromFile(ManifestConfigFile)
}

func ExistsLocal() bool {
	_, err := FS.Stat(ManifestConfigFile)
	return !os.IsNotExist(err)
}

func UpsertOrganizationID(organizationID string) error {
	var mconfig ManifestConfig
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), ManifestConfigFile)
	localConfigFile := ManifestConfigFile
	localConfig, localConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](localConfigFile)
	if localConfigErr == nil {
		mconfig = localConfig
		hasConfig = true
	}
	if !hasConfig {
		globalConfig, globalConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](globalConfigFile)
		if globalConfigErr == nil {
			mconfig = globalConfig
			hasConfig = true
		}
	}

	if !hasConfig {
		mc := ManifestConfig{
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

	var mconfig ManifestConfig

	// Try to read existing global config
	existingConfig, err := readAndUnmarshalConfigJSON[ManifestConfig](globalConfigFile)
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

func UpsertProjectID(projectID string) error {
	var mconfig ManifestConfig
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), ManifestConfigFile)
	localConfigFile := ManifestConfigFile

	localConfig, localConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](localConfigFile)
	if localConfigErr == nil {
		mconfig = localConfig
		hasConfig = true
	}

	if !hasConfig {
		globalConfig, globalConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](globalConfigFile)
		if globalConfigErr == nil {
			mconfig = globalConfig
			hasConfig = true
		}
	}

	if !hasConfig {
		mc := ManifestConfig{
			ProjectId:      &projectID,
			AppName:        nil,
			AppId:          nil,
			AppAlternateId: nil,
			OrganizationId: "",
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

	mconfig.ProjectId = &projectID

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

func UpsertGlobalProjectID(projectID string) error {
	globalDir := GetGlobalDirectory()
	globalConfigFile := filepath.Join(globalDir, ManifestConfigFile)

	var mconfig ManifestConfig

	// Try to read existing global config
	existingConfig, err := readAndUnmarshalConfigJSON[ManifestConfig](globalConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "Error reading global config file")
	}

	// If config exists, use it; otherwise, create a new one
	if err == nil {
		mconfig = existingConfig
	}

	// Update or set the ProjectId
	mconfig.ProjectId = &projectID

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

func UpsertLocalManifestSecret(secret ManifestSecret) error {
	localSecretFile := ManifestSecretFile

	jsonData, err := json.MarshalIndent(secret, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Error encoding JSON")
	}

	err = FS.WriteFile(localSecretFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", localSecretFile)
	}

	return nil
}
