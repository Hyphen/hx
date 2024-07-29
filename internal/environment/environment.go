package environment

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Hyphen/cli/internal/environment/envvars"
	"github.com/Hyphen/cli/internal/environment/infrastructure/envapi"
	"github.com/Hyphen/cli/internal/secretkey"
)

var repository Repository = envapi.New()

func SetRepository(repo Repository) {
	repository = repo
}

type EnvironmentHandler interface {
	EncryptEnvironmentVars(file string) (string, error)
	DecryptEnvironmentVars(env string) ([]string, error)
	DecryptedEnvironmentVarsIntoAFile(env, fileName string) (string, error)
	GetEncryptedEnvironmentVars(env string) (string, error)
	UploadEncryptedEnvironmentVars(env string, envData envvars.EnvironmentVarsData) error
	ListEnvironments(pageSize, pageNum int) ([]envvars.EnvironmentInformation, error)
	SecretKey() secretkey.SecretKeyer
}

var EnvConfigFile = ".hyphen-env-key"

type Environment struct {
	secretKey  secretkey.SecretKeyer
	repository Repository
	config     Config
}

type Config struct {
	AppName   string `toml:"app_name"`
	AppId     string `toml:"app_id"`
	SecretKey string `toml:"secret_key"`
}

func Restore() EnvironmentHandler {
	return RestoreFromFile(EnvConfigFile)
}

func RestoreFromFile(file string) EnvironmentHandler {
	config := Config{}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		fmt.Println("You must init the environment with 'env init'")
		os.Exit(1)
	}

	_, err := toml.DecodeFile(file, &config)
	if err != nil {
		fmt.Println("Error decoding TOML file:", err)
		os.Exit(1)
	}

	return &Environment{
		secretKey:  secretkey.FromBase64(config.SecretKey),
		repository: repository,
		config:     config,
	}
}

func Initialize(appName, appId string) *Environment {
	config := Config{
		AppName:   appName,
		AppId:     appId,
		SecretKey: secretkey.New().Base64(),
	}

	env := &Environment{
		secretKey:  secretkey.FromBase64(config.SecretKey),
		repository: repository,
		config:     config,
	}

	if err := env.repository.Initialize(appName, appId); err != nil {
		if customErr, ok := err.(*envapi.Error); ok {
			fmt.Println("Error:", customErr.UserMessage)
		} else {
			fmt.Println("Error:", err.Error())
		}
		os.Exit(1)
	}

	file, err := os.Create(EnvConfigFile)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	defer file.Close()

	enc := toml.NewEncoder(file)
	if err := enc.Encode(config); err != nil {
		fmt.Println("Error encoding TOML:", err)
		os.Exit(1)
	}

	return env

}

func (e *Environment) SecretKey() secretkey.SecretKeyer {
	return e.secretKey
}

func (e *Environment) GetEncryptedEnvironmentVars(env string) (string, error) {
	env, err := GetEnvName(env)
	if err != nil {
		return "", err
	}
	return e.repository.GetEncryptedVariables(env, e.config.AppId)
}

func (e *Environment) UploadEncryptedEnvironmentVars(env string, envData envvars.EnvironmentVarsData) error {
	env, err := GetEnvName(env)
	if err != nil {
		return err
	}

	if err := envData.EncryptData(e.secretKey); err != nil {
		return err
	}

	if err := e.repository.UploadEnvVariable(env, e.config.AppId, envData); err != nil {
		return err
	}

	return nil
}

func (e *Environment) DecryptEnvironmentVars(env string) ([]string, error) {
	env, err := GetEnvName(env)
	if err != nil {
		return []string{}, err
	}
	envVariables, err := e.GetEncryptedEnvironmentVars(env)
	if err != nil {
		return []string{}, err
	}
	decrypted, err := e.secretKey.Decrypt(envVariables)
	if err != nil {
		return []string{}, nil
	}
	variables := strings.Split(decrypted, "\n")

	return variables, nil

}

func (e *Environment) DecryptedEnvironmentVarsIntoAFile(env, fileName string) (string, error) {
	env, err := GetEnvName(env)
	if err != nil {
		return "", err
	}

	envVars, err := e.DecryptEnvironmentVars(env)
	if err != nil {
		return "", err
	}

	// Open or create the specified file for writing
	file, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("error creating or opening file: %w", err)
	}
	defer file.Close() // Ensure the file is closed after writing

	for _, envVar := range envVars {
		_, err := file.WriteString(envVar + "\n")
		if err != nil {
			return "", fmt.Errorf("error writing environment variables to file: %w", err)
		}
	}

	return fileName, nil
}

func (e *Environment) EncryptEnvironmentVars(file string) (string, error) {
	// Open the file containing the environment variables
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	// Read the file content
	var envVars []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		envVars = append(envVars, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	// Join the environment variables with newline characters
	envVarsStr := strings.Join(envVars, "\n")

	// Encrypt the concatenated string
	encrypted, err := e.secretKey.Encrypt(envVarsStr)
	if err != nil {
		return "", fmt.Errorf("error encrypting environment variables: %w", err)
	}

	return encrypted, nil
}

func (e *Environment) ListEnvironments(pageSize, pageNum int) ([]envvars.EnvironmentInformation, error) {
	return e.repository.ListEnvironments(e.config.AppId, pageSize, pageNum)
}

func tmpDir() string {
	dir, err := os.MkdirTemp("", "tmp.")
	if err != nil {
		fmt.Printf("Error creating temporary directory: %v", err)
		os.Exit(1)
	}
	return dir
}

func tmpFile() (*os.File, string) {
	file, err := os.CreateTemp("", "tmp.")
	if err != nil {
		fmt.Printf("Error creating temporary file: %v", err)
		os.Exit(1)
	}
	return file, file.Name()
}

func EnsureGitignore() error {
	const gitDirPath = ".git"
	const gitignorePath = ".gitignore"

	// Check if the current directory is a Git project
	if _, err := os.Stat(gitDirPath); os.IsNotExist(err) {
		// If the .git directory does not exist, do nothing and return early
		return nil
	}

	// Check if .gitignore exists
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		// If .gitignore does not exist, create it and add the entry
		file, err := os.Create(gitignorePath)
		if err != nil {
			return fmt.Errorf("error creating .gitignore: %w", err)
		}
		defer file.Close()

		_, err = file.WriteString(EnvConfigFile + "\n")
		if err != nil {
			return fmt.Errorf("error writing to .gitignore: %w", err)
		}
		return nil
	}

	// Open the existing .gitignore file for reading and appending
	file, err := os.OpenFile(gitignorePath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("error opening .gitignore: %w", err)
	}
	defer file.Close()

	// Check if the entry already exists in .gitignore
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == EnvConfigFile {
			// Entry already exists in .gitignore
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .gitignore: %w", err)
	}

	// If the entry doesn't exist, add it to the file
	_, err = file.WriteString(EnvConfigFile + "\n")
	if err != nil {
		return fmt.Errorf("error writing to .gitignore: %w", err)
	}

	return nil
}

func ConfigExists() bool {
	_, err := os.Stat(EnvConfigFile)
	return !os.IsNotExist(err)
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

		return "", fmt.Errorf("you are using unpermitted characters. A valid env name can only contain lowercase letters, numbers, hyphens, and underscores. Suggested valid name: %s", suggested)
	}

	return name, nil
}

func CheckAppId(appId string) error {
	validRegex := regexp.MustCompile("^[a-z0-9-_]+$")
	if !validRegex.MatchString(appId) {
		suggested := strings.ToLower(appId)
		suggested = strings.ReplaceAll(suggested, " ", "-")
		suggested = regexp.MustCompile("[^a-z0-9-_]").ReplaceAllString(suggested, "-")
		suggested = regexp.MustCompile("-+").ReplaceAllString(suggested, "-")
		suggested = strings.Trim(suggested, "-")

		return fmt.Errorf("you are using unpermitted characters. A valid env name can only contain lowercase letters, numbers, hyphens, and underscores. Suggested valid name: %s", suggested)
	}

	return nil
}

func GetEnvFileByEnvironment(environment string) string {
	if environment == "" || environment == "default" {
		return ".env"
	}
	return fmt.Sprintf(".env.%s", strings.ToLower(environment))
}
