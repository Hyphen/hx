package manifest

import (
	"encoding/json"
	"os"

	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
)

var ManifestConfigFile = ".hyphen-manifest-key.json"

type Manifest struct {
	AppName        string `json:"app_name"`
	AppId          string `json:"app_id"`
	AppAlternateId string `json:"app_alternate_id"`
	SecretKey      string `json:"secret_key"`
}

func (m *Manifest) GetSecretKey() *secretkey.SecretKey {
	return secretkey.FromBase64(m.SecretKey)
}

func Initialize(organizationId, appName, appID, appAlternateId string) (Manifest, error) {
	sk, err := secretkey.New()
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Failed to create new secret key")
	}

	m := Manifest{
		AppName:        appName,
		AppId:          appID,
		AppAlternateId: appAlternateId,
		SecretKey:      sk.Base64(),
	}

	jsonData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Error encoding JSON")
	}

	err = os.WriteFile(ManifestConfigFile, jsonData, 0644)
	if err != nil {
		return Manifest{}, errors.Wrapf(err, "Error writing file: %s", ManifestConfigFile)
	}

	return m, nil
}

func RestoreFromFile(file string) (Manifest, error) {
	m := Manifest{}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return m, errors.New("You must init the environment with 'env init'")
	}

	jsonData, err := os.ReadFile(file)
	if err != nil {
		return m, errors.Wrap(err, "Error reading JSON file")
	}

	err = json.Unmarshal(jsonData, &m)
	if err != nil {
		return m, errors.Wrap(err, "Error decoding JSON file")
	}

	return m, nil
}

func Restore() (Manifest, error) {
	return RestoreFromFile(ManifestConfigFile)
}

func Exists() bool {
	_, err := os.Stat(ManifestConfigFile)
	return !os.IsNotExist(err)
}
