package environment

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Hyphen/cli/internal/environment/infrastructure/inmemory"
	"github.com/Hyphen/cli/internal/secretkey"
)

type EnviromentHandler interface {
	EncryptEnvironmentVars(file string) (string, error)
	DecryptEnvironmentVars(env string) ([]string, error)
	DecryptedEnviromentVarsIntoAFile(env, fileName string) (string, error)

	GetEncryptedEnviromentVars(env string) (string, error)
	UploadEncryptedEnviromentVars(env, envVars string) error
	SourceEnviromentVars(env string) error
	SecretKey() secretkey.SecretKeyer
}

var EnvConfigFile = ".hyphen-env-key"

type Enviroment struct {
	secretKey  secretkey.SecretKeyer
	repository Repository
}

type Config struct {
	AppName   string `toml:"app_name"`
	SecretKey string `toml:"secret_key"`
}

func Restore() EnviromentHandler {
	return RestoreFromFile(EnvConfigFile)
}

func RestoreFromFile(file string) EnviromentHandler {
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

	return &Enviroment{
		secretKey:  secretkey.FromBase64(config.SecretKey),
		repository: inmemory.New(),
	}
}

func Initialize(appName string) *Enviroment {
	config := Config{
		AppName:   appName,
		SecretKey: secretkey.New().Base64(),
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

	return &Enviroment{
		secretKey:  secretkey.FromBase64(config.SecretKey),
		repository: inmemory.New(),
	}

}

func (e *Enviroment) SecretKey() secretkey.SecretKeyer {
	return e.secretKey
}

func (e *Enviroment) GetEncryptedEnviromentVars(env string) (string, error) {
	return e.repository.GetEncryptedVariables(env)
}

func (e *Enviroment) UploadEncryptedEnviromentVars(env, fileContent string) error {
	encryptedVars, err := e.secretKey.Encrypt(fileContent)
	if err != nil {
		return err
	}

	if err = e.repository.UploadEnvVariable(env, encryptedVars); err != nil {
		return err
	}

	return nil
}

func (e *Enviroment) DecryptEnvironmentVars(env string) ([]string, error) {
	envVariables, err := e.GetEncryptedEnviromentVars(env)
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

func (e *Enviroment) DecryptedEnviromentVarsIntoAFile(env, fileName string) (string, error) {
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

func (e *Enviroment) EncryptEnvironmentVars(file string) (string, error) {
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

func (e *Enviroment) SourceEnviromentVars(env string) error {
	return nil
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

func GetEnvFileByEnvironment(environment string) string {
	if environment == "" || environment == "default" {
		return ".env"
	}
	return fmt.Sprintf(".env.%s", strings.ToLower(environment))
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
