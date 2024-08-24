package env

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
)

// EnvironmentVarsData represents environment variables data.
type EnvironmentVarsData struct {
	Size           string `json:"size"`
	CountVariables int    `json:"countVariables"`
	Data           string `json:"data"`
}

// New processes the environment variables from the given file.
func New(fileName string) (EnvironmentVarsData, error) {
	var data EnvironmentVarsData

	file, err := os.Open(fileName)
	if err != nil {
		return data, errors.Wrap(err, "Failed to open environment file")
	}
	defer file.Close()

	var contentBuilder strings.Builder
	scanner := bufio.NewScanner(file)
	countVariables := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			contentBuilder.WriteString(line + "\n")
			countVariables++
		}
	}

	if err := scanner.Err(); err != nil {
		return data, errors.Wrap(err, "Error scanning environment file")
	}

	content := contentBuilder.String()
	data.Size = strconv.Itoa(len(content)) + " bytes"
	data.CountVariables = countVariables
	data.Data = content

	return data, nil
}

func (e *EnvironmentVarsData) EncryptData(key secretkey.SecretKeyer) (string, error) {
	encryptData, err := key.Encrypt(e.Data)
	if err != nil {
		return "", errors.Wrap(err, "Failed to encrypt environment data")
	}
	return encryptData, nil
}

func (e *EnvironmentVarsData) DecryptData(key secretkey.SecretKeyer) (string, error) {
	decryptedData, err := key.Decrypt(e.Data)
	if err != nil {
		return "", errors.Wrap(err, "Failed to decrypt environment data")
	}
	return decryptedData, nil
}

func (e *EnvironmentVarsData) ListDecryptedVars(key secretkey.SecretKeyer) ([]string, error) {
	decryptedData, err := key.Decrypt(e.Data)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to decrypt environment variables")
	}
	return strings.Split(decryptedData, "\n"), nil
}

func (e *EnvironmentVarsData) DecryptVarsAndSaveIntoFile(fileName string, key secretkey.SecretKeyer) (string, error) {
	file, err := os.Create(fileName)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create or open file for saving decrypted variables")
	}
	defer file.Close()

	decryptedVarList, err := e.ListDecryptedVars(key)
	if err != nil {
		return "", errors.Wrap(err, "Failed to list decrypted variables")
	}

	for _, envVar := range decryptedVarList {
		_, err := file.WriteString(envVar + "\n")
		if err != nil {
			return "", errors.Wrap(err, "Failed to write environment variables to file")
		}
	}

	return fileName, nil
}

type EnvironmentInformation struct {
	Size           string `json:"size"`
	CountVariables int    `json:"countVariables"`
	Data           string `json:"data"`
	AppId          string `json:"appId"`
	EnvId          string `json:"envId"`
}

func (e *EnvironmentInformation) ToEnvironmentVarsData() EnvironmentVarsData {
	return EnvironmentVarsData{
		Size:           e.Size,
		CountVariables: e.CountVariables,
		Data:           e.Data,
	}
}

func GetEnvName(env string) (string, error) {
	if env == "" {
		return "default", nil
	}

	// Convert to lowercase
	name := strings.ToLower(env)

	// Check for unpermitted characters
	validRegex := regexp.MustCompile("^[a-z0-9-_]+$")
	if !validRegex.MatchString(name) {
		// Create a suggested valid name
		suggested := strings.ReplaceAll(name, " ", "-")
		suggested = regexp.MustCompile("[^a-z0-9-_]").ReplaceAllString(suggested, "-")
		suggested = regexp.MustCompile("-+").ReplaceAllString(suggested, "-")
		suggested = strings.Trim(suggested, "-")

		return "", errors.Wrapf(nil, "Invalid environment name. A valid env name can only contain lowercase letters, numbers, hyphens, and underscores. Suggested valid name: %s", suggested)
	}

	return name, nil
}
