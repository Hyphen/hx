package manifest

import (
	"encoding/json"
	"fmt"
	"os"

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
	ManifestConfigFile = ".hyphen-manifest-key.json"
	ManifestSecretFile = ".hyphen-manifest-secret-key.json"
)

type ManifestConfig struct {
	AppName        string `json:"app_name"`
	AppId          string `json:"app_id"`
	AppAlternateId string `json:"app_alternate_id"`
	OrganisationId string `json:"organisation_id"`
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

func Initialize(organizationId, appName, appID, appAlternateId string) (Manifest, error) {
	sk, err := secretkey.New()
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Failed to create new secret key")
	}

	mc := ManifestConfig{
		AppName:        appName,
		AppId:          appID,
		AppAlternateId: appAlternateId,
		OrganisationId: organizationId,
	}
	jsonData, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Error encoding JSON")
	}
	err = os.WriteFile(ManifestConfigFile, jsonData, 0644)
	if err != nil {
		return Manifest{}, errors.Wrapf(err, "Error writing file: %s", ManifestConfigFile)
	}

	ms := ManifestSecret{
		SecretKey: sk.Base64(),
	}
	jsonData, err = json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Error encoding JSON")
	}
	err = os.WriteFile(ManifestSecretFile, jsonData, 0644)
	if err != nil {
		return Manifest{}, errors.Wrapf(err, "Error writing file: %s", ManifestSecretFile)
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

	globalConfigFile := fmt.Sprintf("%s/%s", currentConfigProvider.GetConfigDirectory(), manifestConfigFile)
	globalConfig, globalConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](globalConfigFile)
	if globalConfigErr == nil {
		mconfig = globalConfig
		hasConfig = true
	}

	globalSecretFile := fmt.Sprintf("%s/%s", currentConfigProvider.GetConfigDirectory(), manifestSecretFile)
	globalSecret, globalSecretErr := readAndUnmarshalConfigJSON[ManifestSecret](globalSecretFile)
	if globalSecretErr == nil {
		secret = globalSecret
		hasSecret = true
	}

	localConfig, localConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](manifestConfigFile)
	if localConfigErr == nil {
		mconfig = mergeConfigs(mconfig, localConfig)
		hasConfig = true
	}

	localSecret, localSecretErr := readAndUnmarshalConfigJSON[ManifestSecret](manifestSecretFile)
	if localSecretErr == nil {
		secret = mergeSecrets(secret, localSecret)
		hasSecret = true
	}

	if !hasConfig ||
		mconfig.AppId == "" ||
		mconfig.AppName == "" ||
		mconfig.AppAlternateId == "" {
		return Manifest{}, errors.New("No valid configuration found (neither global nor local)")
	}
	if !hasSecret {
		return Manifest{}, errors.New("No valid secret found (neither global nor local)")
	}

	return Manifest{
		ManifestConfig: mconfig,
		ManifestSecret: secret,
	}, nil
}

func mergeConfigs(base, override ManifestConfig) ManifestConfig {
	merged := base

	if override.AppName != "" {
		merged.AppName = override.AppName
	}
	if override.AppId != "" {
		merged.AppId = override.AppId
	}
	if override.AppAlternateId != "" {
		merged.AppAlternateId = override.AppAlternateId
	}
	if override.OrganisationId != "" {
		merged.OrganisationId = override.OrganisationId
	}

	return merged
}

func mergeSecrets(base, override ManifestSecret) ManifestSecret {
	merged := base

	if override.SecretKey != "" {
		merged.SecretKey = override.SecretKey
	}

	return merged
}

func readAndUnmarshalConfigJSON[T any](filename string) (T, error) {
	var result T

	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return result, errors.Wrapf(err, "Error decoding JSON file: %s", filename)
	}

	return result, nil
}

func Restore() (Manifest, error) {
	return RestoreFromFile(ManifestConfigFile, ManifestSecretFile)
}

func Exists() bool {
	_, err := os.Stat(ManifestConfigFile)
	return !os.IsNotExist(err)
}

func UpdateOrganizationID(organizationID string) error {
	var mconfig ManifestConfig
	var hasConfig bool

	globalConfigFile := fmt.Sprintf("%s/%s", currentConfigProvider.GetConfigDirectory(), ManifestConfigFile)
	localConfigFile := ManifestConfigFile
	localConfig, localConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](localConfigFile)
	if localConfigErr == nil {
		mconfig = localConfig
		hasConfig = true
	}
	fmt.Println(globalConfigFile)
	fmt.Println(localConfigFile)

	if !hasConfig {
		globalConfig, globalConfigErr := readAndUnmarshalConfigJSON[ManifestConfig](globalConfigFile)
		if globalConfigErr == nil {
			mconfig = globalConfig
			hasConfig = true
		}
	}

	if !hasConfig {
		mc := ManifestConfig{
			AppName:        "",
			AppId:          "",
			AppAlternateId: "",
			OrganisationId: organizationID,
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

	mconfig.OrganisationId = organizationID

	configFile := localConfigFile
	if localConfigErr != nil {
		configFile = globalConfigFile
	}

	jsonData, err := json.MarshalIndent(mconfig, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Error encoding JSON")
	}
	err = os.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing file: %s", configFile)
	}

	return nil
}
