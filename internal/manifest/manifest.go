package manifest

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
)

var ManifestConfigFile = ".hyphen-manifest-key"

type Manifest struct {
	ProjectName        string `toml:"project_name"`
	ProjectId          string `toml:"project_id"`
	ProjectAlternateId string `toml:"project_alternate_id"`
	SecretKey          string `toml:"secret_key"`
}

func (m *Manifest) GetSecretKey() *secretkey.SecretKey {
	return secretkey.FromBase64(m.SecretKey)
}

func Initialize(organizationId, projectName, projectID, projectAlternateId string) (Manifest, error) {
	sk, err := secretkey.New()
	if err != nil {
		return Manifest{}, errors.Wrap(err, "Failed to create new secret key")
	}

	m := Manifest{
		ProjectName:        projectName,
		ProjectId:          projectID,
		ProjectAlternateId: projectAlternateId,
		SecretKey:          sk.Base64(),
	}

	file, err := os.Create(ManifestConfigFile)
	if err != nil {
		return Manifest{}, errors.Wrapf(err, "Error creating file: %s", ManifestConfigFile)
	}
	defer file.Close()

	enc := toml.NewEncoder(file)
	if err := enc.Encode(m); err != nil {
		return Manifest{}, errors.Wrap(err, "Error encoding TOML")
	}

	return m, nil
}

func RestoreFromFile(file string) (Manifest, error) {
	m := Manifest{}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return m, errors.New("You must init the environment with 'env init'")
	}

	_, err := toml.DecodeFile(file, &m)
	if err != nil {
		return m, errors.Wrap(err, "Error decoding TOML file")
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
