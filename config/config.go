package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/fsutil"
)

var FS fsutil.FileSystem = fsutil.NewFileSystem()

type CredentialsConfig struct {
	Default Credentials `json:"default"`
}

type Credentials struct {
	HyphenAccessToken  string `json:"hyphen_access_token"`
	HyphenRefreshToken string `json:"hyphen_refresh_token"`
	HypenIDToken       string `json:"hyphen_id_token"`
	ExpiryTime         int64  `json:"expiry_time"`
}

// Filepaths for credential storage
const (
	WindowsConfigPath = "Hyphen"
	UnixConfigPath    = ".hyphen"
	CredentialFile    = "credentials.json"
)

func Init(fs fsutil.FileSystem) {
	FS = fs
}

func GetConfigDirectory() string {
	switch runtime.GOOS {
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

func ensureDir(dirName string) error {
	_, err := FS.Stat(dirName)
	if err != nil {
		if os.IsNotExist(err) {
			return FS.MkdirAll(dirName, 0755)
		}
		return err
	}
	return nil
}

// SaveCredentials stores credentials in a system-dependent location
func SaveCredentials(accessToken, refreshToken, IDToken string, expiryTime int64) error {
	configDir := GetConfigDirectory()
	if err := ensureDir(configDir); err != nil {
		return errors.Wrap(err, "Failed to create configuration directory")
	}

	credentialFilePath := filepath.Join(configDir, CredentialFile)
	credentials := CredentialsConfig{
		Default: Credentials{
			HyphenAccessToken:  accessToken,
			HyphenRefreshToken: refreshToken,
			HypenIDToken:       IDToken,
			ExpiryTime:         expiryTime,
		},
	}

	jsonData, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Failed to marshal credentials to JSON")
	}

	// Write the credentials to the file
	if err := FS.WriteFile(credentialFilePath, jsonData, 0644); err != nil {
		return errors.Wrap(err, "Failed to save credentials")
	}

	return nil
}

// LoadCredentials retrieves credentials from a configuration file
func LoadCredentials() (CredentialsConfig, error) {
	configDir := GetConfigDirectory()
	credentialFilePath := filepath.Join(configDir, CredentialFile)

	data, err := FS.ReadFile(credentialFilePath)
	if err != nil {
		return CredentialsConfig{}, fmt.Errorf("Failed to read credentials file: %w", err)
	}

	var config CredentialsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return CredentialsConfig{}, fmt.Errorf("Failed to parse credentials file: %w", err)
	}

	return config, nil
}
