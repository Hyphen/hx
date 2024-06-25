package environment

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Hyphen/cli/internal/secretkey"
)

type EnviromentHandler interface {
	EncryptEnvironmentVars(file string) (string, error)
	DecryptEnvironmentVars(env string) ([]string, error)
	DecryptedEnviromentVarsIntoAFile(env, fileName string) (string, error)

	GetEncryptedEnviromentVars(env string) (string, error)
	UploadEncryptedEnviromentVars(env string) error
	SourceEnviromentVars(env string) error
	SecretKey() secretkey.SecretKeyer
}

var EnvConfigFile = ".hyphen-env-key"

type Enviroment struct {
	secretKey secretkey.SecretKeyer
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
	_, err := toml.DecodeFile(file, &config)
	if err != nil {
		fmt.Println("Error decoding TOML file:", err)
		os.Exit(1)
	}

	return &Enviroment{
		secretKey: secretkey.FromBase64(config.SecretKey),
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
		secretKey: secretkey.FromBase64(config.SecretKey),
	}

}

func (e *Enviroment) SecretKey() secretkey.SecretKeyer {
	return e.secretKey
}

func (e *Enviroment) GetEncryptedEnviromentVars(env string) (string, error) {
	//TODO get the ecripted from the correct env viroments
	return "A0ypRST4-Bfd6Z9zGwzO6tSQagByUtlHFZ42", nil
}

func (e *Enviroment) UploadEncryptedEnviromentVars(env string) error {
	//TODO upload the env to the correct env
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
	if environment == "" {
		return ".env"
	}
	return fmt.Sprintf(".env.%s", strings.ToLower(environment))
}
