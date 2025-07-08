package env

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
)

func New(fileName string) (models.Env, error) {
	var data models.Env

	content, err := os.ReadFile(fileName)
	if err != nil {
		return data, errors.Wrapf(err, "Failed to open environment file '%s'", fileName)
	}

	contentStr := string(content)

	countVariables := countEnvVars(contentStr)

	data.Size = strconv.Itoa(len(contentStr)) + " bytes"
	data.CountVariables = countVariables
	data.Data = contentStr
	data.Version = nil
	data.ProjectEnv = nil

	return data, nil
}

func countEnvVars(content string) int {
	count := 0
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if IsEnvVar(line) {
			count++
		}
	}
	return count
}

func IsEnvVar(line string) bool {
	return line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=")
}

func NewWithEncryptedData(fileName string, key secretkey.SecretKeyer) (models.Env, error) {
	env, err := New(fileName)
	if err != nil {
		return models.Env{}, err
	}
	data, err := env.EncryptData(key)
	env.Data = data
	return env, nil
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
