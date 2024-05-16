package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
	"runtime"
)

type CredentialsConfig struct {
	Default Credentials `toml:"default"`
}

type Credentials struct {
	HyphenAccessToken string `toml:"hyphen_access_token"`
}

// Filepaths for credential storage
const (
	WindowsConfigPath = "Hyphen"
	UnixConfigPath    = ".hyphen"
	CredentialFile    = "credentials.toml"
)

type Environment interface {
	GetConfigDirectory() string
	EnsureDir(dirName string) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadFile(path string) ([]byte, error)
	GetGOOS() string
}

type systemEnvironment struct{}

func (se *systemEnvironment) GetGOOS() string {
	return runtime.GOOS
}

func (se *systemEnvironment) GetConfigDirectory() string {
	switch se.GetGOOS() {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), WindowsConfigPath)
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error retrieving home directory:", err)
			return ""
		}
		return filepath.Join(home, UnixConfigPath)
	}
}

func (se *systemEnvironment) EnsureDir(dirName string) error {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err := os.MkdirAll(dirName, 0755)
		if err != nil {
			return fmt.Errorf("Error creating directory: %w", err)
		}
	}
	return nil
}

func (se *systemEnvironment) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (se *systemEnvironment) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

var Env Environment = &systemEnvironment{}

// SaveCredentials stores credentials in a system-dependent location
func SaveCredentials(username, password string) {
	configDir := Env.GetConfigDirectory()
	if err := Env.EnsureDir(configDir); err != nil {
		fmt.Println(err)
		return
	}

	credentialFilePath := filepath.Join(configDir, CredentialFile)
	credentialsContent := fmt.Sprintf("[default]\nhyphen_access_token=\"%s:%s\"", username, password)

	// Write the credentials to the file
	if err := Env.WriteFile(credentialFilePath, []byte(credentialsContent), 0644); err != nil {
		fmt.Println("Error writing credentials to file:", err)
		return
	}

	fmt.Println("Credentials saved successfully to", credentialFilePath)
}

// GetCredentials retrieves credentials from a configuration file
func GetCredentials() (CredentialsConfig, error) {
	configDir := Env.GetConfigDirectory()
	credentialFilePath := filepath.Join(configDir, CredentialFile)

	// Read file using Environment's ReadFile method
	data, err := Env.ReadFile(credentialFilePath)
	if err != nil {
		return CredentialsConfig{}, fmt.Errorf("failed to read file: %w", err)
	}

	var config CredentialsConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return CredentialsConfig{}, fmt.Errorf("failed to decode credentials: %w", err)
	}

	return config, nil
}
