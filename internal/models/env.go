package models

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"

	"github.com/Hyphen/cli/pkg/errors"
)

type Env struct {
	Size           string                       `json:"size"`
	CountVariables int                          `json:"countVariables"`
	Data           string                       `json:"data"`
	ID             *string                      `json:"id,omitempty"`
	Version        *int                         `json:"version,omitempty"`
	ProjectEnv     *ProjectEnvironmentReference `json:"projectEnvironment,omitempty"`
	SecretKeyId    *int64                       `json:"secretKeyId,omitempty"`
	Published      *time.Time                   `json:"published,omitempty"`
}

// TODO -- some of this stuff seems like it should probably just live in the env service, right? That can then hold the secret keyer?

// HashData returns the SHA256 hash of the environment data.
func HashData(data string) string {
	hash := sha256.New()
	hash.Write([]byte(data))
	hashSum := hash.Sum(nil)
	return hex.EncodeToString(hashSum)
}

func (e *Env) HashData() string {
	return HashData(e.Data)
}

func (e *Env) EncryptData(secret Secret) (string, error) {
	encryptData, err := secret.Encrypt(e.Data)
	if err != nil {
		return "", errors.Wrap(err, "Failed to encrypt environment data")
	}
	return encryptData, nil
}

func (e *Env) DecryptData(secret Secret) (string, error) {
	decryptedData, err := secret.Decrypt(e.Data)
	if err != nil {
		return "", errors.Wrap(err, "Failed to decrypt environment data")
	}
	return decryptedData, nil
}

func (e *Env) DecryptVarsAndSaveIntoFile(fileName string, secret Secret) (string, error) {
	file, err := os.Create(fileName)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create or open file for saving decrypted variables")
	}
	defer file.Close()

	decryptedData, err := secret.Decrypt(e.Data)
	if err != nil {
		return "", errors.Wrap(err, "Failed to decrypt environment data")
	}

	_, err = file.WriteString(decryptedData)
	if err != nil {
		return "", errors.Wrap(err, "Failed to write decrypted data to file")
	}

	return fileName, nil
}
