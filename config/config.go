package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

type CredentialsConfig struct {
	Default Credentials `toml:"default"`
}

type Credentials struct {
	HyphenAccessToken  string `toml:"hyphen_access_token"`
	HyphenRefreshToken string `toml:"hyphen_refresh_token"`
	HypenIDToken       string `toml:"hyphen_id_token"`
	ExpiryTime         int64  `toml:"expiry_time"`
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
func SaveCredentials(accessToken, refreshToken, IDToken string, expiryTime int64) error {
	configDir := Env.GetConfigDirectory()
	if err := Env.EnsureDir(configDir); err != nil {
		return err
	}

	credentialFilePath := filepath.Join(configDir, CredentialFile)
	credentialsContent := fmt.Sprintf(
		"[default]\nhyphen_access_token=\"%s\"\nhyphen_refresh_token=\"%s\"\nhyphen_id_token=\"%s\"\nexpiry_time=%d",
		accessToken, refreshToken, IDToken, expiryTime)

	// Write the credentials to the file
	if err := Env.WriteFile(credentialFilePath, []byte(credentialsContent), 0644); err != nil {
		return fmt.Errorf("Error writing credentials to file: %w", err)
	}

	return nil
}

// LoadCredentials retrieves credentials from a configuration file
func LoadCredentials() (CredentialsConfig, error) {
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
