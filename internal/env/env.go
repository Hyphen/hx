package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
)

type Env struct {
	Size           string              `json:"size"`
	CountVariables int                 `json:"countVariables"`
	Data           string              `json:"data"`
	Version        *int                `json:"version,omitempty"`
	ProjectEnv     *ProjectEnvironment `json:"projectEnvironment,omitempty"`
}
type ProjectEnvironment struct {
	ID          string `json:"id"`
	AlternateID string `json:"alternateId"`
	Name        string `json:"name"`
}

func New(fileName string) (Env, error) {
	var data Env

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
	data.Version = nil
	data.ProjectEnv = nil

	return data, nil
}

func (e *Env) EncryptData(key secretkey.SecretKeyer) (string, error) {
	encryptData, err := key.Encrypt(e.Data)
	if err != nil {
		return "", errors.Wrap(err, "Failed to encrypt environment data")
	}
	return encryptData, nil
}

func (e *Env) DecryptData(key secretkey.SecretKeyer) (string, error) {
	decryptedData, err := key.Decrypt(e.Data)
	if err != nil {
		return "", errors.Wrap(err, "Failed to decrypt environment data")
	}
	return decryptedData, nil
}

func (e *Env) ListDecryptedVars(key secretkey.SecretKeyer) ([]string, error) {
	decryptedData, err := key.Decrypt(e.Data)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to decrypt environment variables")
	}
	return strings.Split(decryptedData, "\n"), nil
}

func (e *Env) DecryptVarsAndSaveIntoFile(fileName string, key secretkey.SecretKeyer) (string, error) {
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

type Environment struct {
	ID           string       `json:"id"`
	AlternateID  string       `json:"alternateId"`
	Name         string       `json:"name"`
	Color        string       `json:"color"`
	Organization Organization `json:"organization"`
	Project      Project      `json:"project"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID          string `json:"id"`
	AlternateID string `json:"alternateId"`
	Name        string `json:"name"`
}

func GetEnvName(env string) (string, error) {
	if env == "" || env == "default" {
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

func GetFileName(env string) (string, error) {
	name, err := GetEnvName(env)
	if err != nil {
		return "", err
	}

	if name == "default" {
		return ".env", nil
	}

	return fmt.Sprintf(".%s.env", name), nil

}

func GetEnvsInDirectory() ([]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get current working directory")
	}

	var envFiles []string

	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".env") {
			relPath, err := filepath.Rel(currentDir, path)
			if err != nil {
				return errors.Wrap(err, "Failed to get relative path")
			}
			envFiles = append(envFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "Failed to walk through directory")
	}

	return envFiles, nil
}

func GetEnvNameFromFile(fileName string) (string, error) {
	if fileName == ".env" {
		return "default", nil
	}

	validRegex := regexp.MustCompile(`^\.?[a-z0-9-_]+\.env$`)
	if !validRegex.MatchString(fileName) {
		return "", errors.Wrapf(nil, "Invalid environment file name. A valid env file name can only contain lowercase letters, numbers, hyphens, and underscores")
	}

	trimmedName := strings.TrimPrefix(fileName, ".")

	envName := strings.TrimSuffix(trimmedName, ".env")

	return envName, nil
}
