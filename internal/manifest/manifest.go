package manifest

import (
	"encoding/json"
	"os"

	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
)

var (
	ManifestConfigFile = ".hyphen-manifest-key.json"
	ManifestSecretFile = ".hyphen-manifest-secret-key.json"
)

type ManifestConfig struct {
	AppName        string `json:"app_name"`
	AppId          string `json:"app_id"`
	AppAlternateId string `json:"app_alternate_id"`
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
	mc, err := readAndUnmarshalConfigJSON[ManifestConfig](manifestConfigFile)
	if err != nil {
		return Manifest{}, err
	}

	ms, err := readAndUnmarshalConfigJSON[ManifestSecret](manifestSecretFile)
	if err != nil {
		return Manifest{}, err
	}

	return Manifest{
		ManifestConfig: mc,
		ManifestSecret: ms,
	}, nil
}

func readAndUnmarshalConfigJSON[T any](filename string) (T, error) {
	var result T

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return result, errors.New("You must init the environment with 'env init'")
	}

	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return result, errors.Wrap(err, "Error reading JSON file")
	}

	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return result, errors.Wrap(err, "Error decoding JSON file")
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
