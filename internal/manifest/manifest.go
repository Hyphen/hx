package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/Hyphen/cli/config"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
)

type configProvider interface {
	GetConfigDirectory() string
}

type defaultConfigProvider struct{}

func (d *defaultConfigProvider) GetConfigDirectory() string {
	return config.GetConfigDirectory()
}

var currentConfigProvider configProvider = &defaultConfigProvider{}

func SetConfigProvider(provider configProvider) {
	currentConfigProvider = provider
}

var (
	ManifestConfigFile = ".hx"
	ManifestSecretFile = ".hxkey"
)

type ManifestConfig struct {
	ProjectName        *string `json:"project_name,omitempty"`
	ProjectId          *string `json:"project_id,omitempty"`
	ProjectAlternateId *string `json:"project_alternate_id,omitempty"`
	AppName            *string `json:"app_name,omitempty"`
	AppId              *string `json:"app_id,omitempty"`
	AppAlternateId     *string `json:"app_alternate_id,omitempty"`
	OrganizationId     string  `json:"organization_id"`
}

type Manifest struct {
	ManifestConfig
	ManifestSecret
}

type ManifestSecret struct {
	SecretKey string `json:"secret_key"`
}

func (m *Manifest) GetSecretKey() *secretkey.SecretKey {
	return secretkey.FromBase64(m.SecretKey)
}

func LocalInitialize(mc ManifestConfig) (Manifest, error) {
	return Initialize(mc, ManifestSecretFile, ManifestConfigFile)
}

func GlobalInitialize(mc ManifestConfig) (Manifest, error) {
	configDirectory := currentConfigProvider.GetConfigDirectory()

	manifestSecretFilePath := fmt.Sprintf("%s/%s", configDirectory, ManifestSecretFile)
	manifestConfigFilePath := fmt.Sprintf("%s/%s", configDirectory, ManifestConfigFile)

	return Initialize(mc, manifestSecretFilePath, manifestConfigFilePath)
}

func Initialize(mc ManifestConfig, secretFile, configFile string) (Manifest, error) {
	sk, err := secretkey.New()
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Failed to create new secret key")
	}

	jsonData, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Error encoding JSON")
	}
	err = os.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		return Manifest{}, errors.Wrapf(err, "Error writing file: %s", configFile)
	}

	ms := ManifestSecret{
		SecretKey: sk.Base64(),
	}
	jsonData, err = json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Error encoding JSON")
	}
	err = os.WriteFile(secretFile, jsonData, 0644)
	if err != nil {
		return Manifest{}, errors.Wrapf(err, "Error writing file: %s", secretFile)
	}

	m := Manifest{
		mc,
		ms,
	}

	return m, nil
}

func RestoreFromFile(manifestConfigFile, manifestSecretFile string) (Manifest, error) {
	var mconfig ManifestConfig
	var secret ManifestSecret
	var hasConfig, hasSecret bool
	var hasOnlyGlobalConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", currentConfigProvider.GetConfigDirectory(), manifestConfigFile)
	globalSecretFile := fmt.Sprintf("%s/%s", currentConfigProvider.GetConfigDirectory(), manifestSecretFile)

	globalConfig, err := readAndUnmarshalConfigJSON[ManifestConfig](globalConfigFile)
	if err == nil {
		mconfig = globalConfig
		hasConfig = true
		hasOnlyGlobalConfig = true
	} else if !os.IsNotExist(err) {
		return Manifest{}, err
	}

	globalSecret, err := readAndUnmarshalConfigJSON[ManifestSecret](globalSecretFile)
	if err == nil {
		secret = globalSecret
		hasSecret = true
		hasOnlyGlobalConfig = false
	} else if !os.IsNotExist(err) {
		return Manifest{}, err
	}

	localConfig, localConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](manifestConfigFile)
	if localConfigErr == nil {
		mergeErr := mergo.Merge(&mconfig, localConfig, mergo.WithOverride)
		if mergeErr != nil {
			return Manifest{}, errors.Wrap(mergeErr, "Error merging your .hx config(s)")
		}
		hasConfig = true
	} else if !os.IsNotExist(localConfigErr) {
		return Manifest{}, localConfigErr
	}

	localSecret, localSecretErr := readAndUnmarshalConfigJSON[ManifestSecret](manifestSecretFile)
	if localSecretErr == nil {
		mergeErr := mergo.Merge(&secret, localSecret, mergo.WithOverride)
		if mergeErr != nil {
			return Manifest{}, errors.Wrap(mergeErr, "Error merging your .hxkey secret(s)")
		}
		hasSecret = true
	} else if !os.IsNotExist(localSecretErr) {
		return Manifest{}, localSecretErr
	}

	if !hasConfig {
		return Manifest{}, errors.New("No valid configuration found (neither global nor local)")
	}

	if !hasSecret && !hasOnlyGlobalConfig {
		return Manifest{}, errors.New("No valid secret found (neither global nor local)")
	}

	return Manifest{
		ManifestConfig: mconfig,
		ManifestSecret: secret,
	}, nil
}

func readAndUnmarshalConfigJSON[T any](filename string) (T, error) {
	var result T

	jsonData, err := os.ReadFile(filename)
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

func ExistsLocal() bool {
	_, err := os.Stat(ManifestConfigFile)
	return !os.IsNotExist(err)
}

func UpsertOrganizationID(organizationID string) error {
	var mconfig ManifestConfig
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", currentConfigProvider.GetConfigDirectory(), ManifestConfigFile)
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
		err = os.WriteFile(globalConfigFile, jsonData, 0644)
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
		return errors.Wrapf(err, "Error decoding JSON file: %s, error: %v", configFile, err)
	}
	err = os.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", configFile)
	}

	return nil
}

func UpsertProjectID(projectID string) error {
	var mconfig ManifestConfig
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", currentConfigProvider.GetConfigDirectory(), ManifestConfigFile)
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
		err = os.WriteFile(globalConfigFile, jsonData, 0644)
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
	err = os.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", configFile)
	}

	return nil
}
