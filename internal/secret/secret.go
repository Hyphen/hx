package secret

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"dario.cat/mergo"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/vinz"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/fsutil"
)

var FS fsutil.FileSystem = fsutil.NewFileSystem()
var vs vinz.VinzServicer

var (
	ManifestSecretFile = ".hxkey"
)

func getVinzService() vinz.VinzServicer {
	if vs == nil {
		vs = vinz.NewService()
	}
	return vs
}

type SecretLocation int

const (
	SecretLocationVinz SecretLocation = iota
	SecretLocationLocal
	SecretLocationNone
)

func NewSecret(secretBase64 string) models.Secret {
	return models.NewSecret(secretBase64)
}

func LoadOrInitializeSecret(organizationId, projectIdOrAlternateId string) (models.Secret, SecretLocation, error) {
	// First, attempt to load the secret
	secret, location, err := LoadSecret(organizationId, projectIdOrAlternateId)
	if location == SecretLocationNone {
		initToLocation := SecretLocationVinz
		if flags.LocalSecret {
			initToLocation = SecretLocationLocal
		}

		secret, err = InitializeSecret(organizationId, projectIdOrAlternateId, initToLocation, ManifestSecretFile)
		return secret, initToLocation, err
	}

	if flags.LocalSecret && location == SecretLocationVinz {
		// warn
		cprint.Warning("This project already has remotely configured secrets. Ignoring --localSecret flag.")
	}

	return secret, location, err
}

func LoadSecret(organizationId, projectIdOrAlternateId string) (models.Secret, SecretLocation, error) {
	// Always default to looking in Vinz first, unless there is a LocalSecret flag.
	if !flags.LocalSecret {
		secret, err := getVinzService().GetKey(organizationId, projectIdOrAlternateId)
		if err == nil {
			return models.Secret{
				SecretKeyId:     secret.SecretKeyId,
				Base64SecretKey: secret.SecretKey,
			}, SecretLocationVinz, nil
		}
	}

	if _, err := os.Stat(ManifestSecretFile); err == nil {
		secret, err := restoreSecretFromFile(ManifestSecretFile)
		if err == nil {
			return secret, SecretLocationLocal, nil
		}
	}

	return models.Secret{}, SecretLocationNone, nil
}

func InitializeSecret(organizationId, projectIdOrAlternateId string, secretLocation SecretLocation, secretFile string) (models.Secret, error) {
	ms, err := models.GenerateSecret()
	if err != nil {
		return models.Secret{}, errors.Wrap(err, "Failed to create new secret key")
	}

	jsonData, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return models.Secret{}, errors.Wrap(err, "Error encoding JSON")
	}

	switch secretLocation {
	case SecretLocationLocal:
		err = FS.WriteFile(secretFile, jsonData, 0644)
		if err != nil {
			return models.Secret{}, errors.Wrapf(err, "Error writing file: %s", secretFile)
		}
	case SecretLocationVinz:
		_, err := getVinzService().SaveKey(organizationId, projectIdOrAlternateId, vinz.Key{
			SecretKeyId: ms.SecretKeyId,
			SecretKey:   ms.Base64(),
		})
		if err != nil {
			return models.Secret{}, errors.Wrap(err, "Failed to save secret key")
		}
	default:
		return ms, fmt.Errorf("specified secret location must be local or vinz")
	}

	return ms, nil
}

func restoreSecretFromFile(manifestSecretFile string) (models.Secret, error) {
	monorepoSecret, err := RestoreSecretFromMonorepo()
	if err == nil {
		return monorepoSecret, nil
	}

	var secret models.Secret
	//var hasSecret bool

	globalSecretFile := fmt.Sprintf("%s/%s", GetGlobalDirectory(), manifestSecretFile)

	globalSecret, err := readAndUnmarshalConfigJSON[models.Secret](globalSecretFile)
	if err == nil {
		secret = globalSecret
		//hasSecret = true
	} else if !os.IsNotExist(err) {
		return models.Secret{}, err
	}

	localSecret, localSecretErr := readAndUnmarshalConfigJSON[models.Secret](manifestSecretFile)
	if localSecretErr == nil {
		mergeErr := mergo.Merge(&secret, localSecret, mergo.WithOverride)
		if mergeErr != nil {
			return models.Secret{}, errors.Wrap(mergeErr, "Error merging your .hxkey secret(s)")
		}
		//hasSecret = true
	} else if !os.IsNotExist(localSecretErr) {
		return models.Secret{}, localSecretErr
	}

	// if !hasSecret {
	// 	return Secret{}, errors.New("No valid .hxkey found (neither global, local, nor monorepo). Please init an app using `hx init`")
	// }

	return secret, nil
}

func RestoreSecretFromMonorepo() (models.Secret, error) {
	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return models.Secret{}, errors.Wrap(err, "Failed to get current working directory")
	}

	// Keep traversing up until we find a monorepo config or hit root
	for {

		// TODO: we need to revisit this logic, previously
		// this would rely on config to find the key file.

		// Check for .hx file in current directory
		// configPath := filepath.Join(currentDir, ManifestConfigFile)
		// config, err := readAndUnmarshalConfigJSON[Config](configPath)

		// // If we can read the config and it's a monorepo
		// if err == nil && config.IsMonorepoProject() {
		// 	// Look for .hxkey in the same directory
		// 	secretPath := filepath.Join(currentDir, ManifestSecretFile)
		// 	secret, err := readAndUnmarshalConfigJSON[Secret](secretPath)
		// 	if err == nil {
		// 		return secret, nil
		// 	}
		// 	return Secret{}, errors.Wrapf(err, "Found monorepo config at %s but failed to read secret file", currentDir)
		// }

		// Look for .hxkey in the same directory
		secretPath := filepath.Join(currentDir, ManifestSecretFile)
		secret, err := readAndUnmarshalConfigJSON[models.Secret](secretPath)
		if err == nil {
			return secret, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)

		// Check if we've hit the root directory
		if parentDir == currentDir {
			return models.Secret{}, errors.New("No monorepo configuration found in parent directories")
		}

		currentDir = parentDir
	}
}

func UpsertLocalSecret(secret models.Secret) error {
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

func RotateSecret() (models.Secret, error) {
	newSecret, err := models.GenerateSecret()
	if err != nil {
		return newSecret, errors.Wrap(err, "Failed to generate new secret key")
	}

	organizationId, err := flags.GetOrganizationID()
	if err != nil {
		return newSecret, errors.Wrap(err, "Failed to get organization ID")
	}

	projectId, err := flags.GetProjectID()
	if err != nil {
		return newSecret, errors.Wrap(err, "Failed to get project ID")
	}

	// If there's a secret in Vinz, we're using vinz for this project. Do not use local secret.
	_, err = getVinzService().GetKey(organizationId, projectId)
	if err == nil {
		// rotate in vinz
		_, err = getVinzService().SaveKey(organizationId, projectId, vinz.Key{
			SecretKeyId: newSecret.SecretKeyId,
			SecretKey:   newSecret.Base64(),
		})
		if err != nil {
			return newSecret, errors.Wrap(err, "Failed to save new key to Vinz")
		}

		return newSecret, nil
	}

	// rotate locally
	err = UpsertLocalSecret(newSecret)
	if err != nil {
		return newSecret, errors.Wrap(err, "Failed to save new secret locally")
	}

	return newSecret, nil
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

func GetGlobalDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error retrieving home directory:", err)
		return ""
	}
	return home
}
