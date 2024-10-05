package env

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
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

const metadataHeader = `######################################
#              Metadata              #
#                                    #`

const metadataFooter = `######################################`

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
	versionFound := false
	inMetadataSection := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "######################################") {
			if inMetadataSection {
				inMetadataSection = false
			} else {
				inMetadataSection = true
			}
			continue
		}
		if inMetadataSection && strings.HasPrefix(line, "# ENV_METADATA_VERSION=") {
			versionStr := strings.TrimPrefix(line, "# ENV_METADATA_VERSION=")
			versionStr = strings.TrimSpace(versionStr)
			versionStr = strings.TrimSuffix(versionStr, "#")
			versionStr = strings.TrimSpace(versionStr) // Add this line to remove any trailing spaces
			version, err := strconv.Atoi(versionStr)
			if err == nil {
				data.Version = &version
				versionFound = true
			} else {
				return data, errors.Wrap(err, "Error parsing version number")
			}
		} else if !inMetadataSection {
			contentBuilder.WriteString(line + "\n")
			if strings.Contains(line, "=") {
				countVariables++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return data, errors.Wrap(err, "Error scanning environment file")
	}

	content := contentBuilder.String()
	data.Size = strconv.Itoa(len(content)) + " bytes"
	data.CountVariables = countVariables
	data.Data = content

	if !versionFound {
		defaultVersion := 1
		data.Version = &defaultVersion
		fmt.Printf("Version not found, defaulting to: %d\n", *data.Version)
	}

	data.ProjectEnv = nil

	return data, nil
}

func (e *Env) EncryptData(key secretkey.SecretKeyer) (string, error) {
	if e.Version == nil {
		defaultVersion := 1
		e.Version = &defaultVersion
	}
	metadataSection := fmt.Sprintf("%s\n# ENV_METADATA_VERSION=%-15d#\n%s\n", metadataHeader, *e.Version, metadataFooter)
	dataWithMetadata := metadataSection + e.Data
	encryptData, err := key.Encrypt(dataWithMetadata)
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

	// Remove the metadata section if present
	lines := strings.Split(decryptedData, "\n")
	startIndex := -1
	endIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "######################################") {
			if startIndex == -1 {
				startIndex = i
			} else {
				endIndex = i
				break
			}
		}
	}

	if startIndex != -1 && endIndex != -1 && endIndex > startIndex {
		return strings.Join(append(lines[:startIndex], lines[endIndex+1:]...), "\n"), nil
	}

	return decryptedData, nil
}

func (e *Env) ListDecryptedVars(key secretkey.SecretKeyer) ([]string, error) {
	decryptedData, err := e.DecryptData(key)
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

	if e.Version == nil {
		defaultVersion := 1
		e.Version = &defaultVersion
	}

	// Write metadata section
	_, err = file.WriteString(fmt.Sprintf("%s\n# ENV_METADATA_VERSION=%-15d#\n%s\n\n", metadataHeader, *e.Version, metadataFooter))
	if err != nil {
		return "", errors.Wrap(err, "Failed to write metadata to file")
	}

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

func (e *Env) SaveToFile(fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrap(err, "Failed to create environment file")
	}
	defer file.Close()

	if e.Version == nil {
		defaultVersion := 1
		e.Version = &defaultVersion
	}

	// Write metadata section
	_, err = file.WriteString(fmt.Sprintf("%s\n# ENV_METADATA_VERSION=%-15d#\n%s\n\n", metadataHeader, *e.Version, metadataFooter))
	if err != nil {
		return errors.Wrap(err, "Failed to write metadata to file")
	}

	// Write the rest of the data
	_, err = file.WriteString(e.Data)
	if err != nil {
		return errors.Wrap(err, "Failed to write environment data to file")
	}

	return nil
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

func (e *Env) IncrementVersion() {
	if e.Version == nil {
		defaultVersion := 1
		e.Version = &defaultVersion
	} else {
		newVersion := *e.Version + 1
		e.Version = &newVersion
	}
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

	return fmt.Sprintf(".env.%s", name), nil
}

func GetEnvsInDirectory() ([]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get current working directory")
	}

	var envFiles []string

	files, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read directory")
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), ".env") && !strings.HasSuffix(file.Name(), ".local") {
			envFiles = append(envFiles, file.Name())
		}
	}

	return envFiles, nil
}

func GetEnvNameFromFile(fileName string) (string, error) {
	if fileName == ".env" {
		return "default", nil
	}

	validRegex := regexp.MustCompile(`^\.env\.?[a-z0-9-_]+$`)
	if !validRegex.MatchString(fileName) {
		return "", errors.Wrapf(nil, "Invalid .env file name encountered: '%s'. A valid .env file name can only contain lowercase letters, numbers, hyphens, and underscores", fileName)
	}

	envName := strings.TrimPrefix(fileName, ".env.")

	return envName, nil
}

func GetEnvironmentID() (string, error) {
	if flags.EnvironmentFlag != "" {
		envName, err := GetEnvName(flags.EnvironmentFlag)
		if err != nil {
			return "", errors.Wrap(err, fmt.Sprintf("environment '%s' is not valid", flags.EnvironmentFlag))
		}
		return envName, nil
	}

	return "default", nil
}
