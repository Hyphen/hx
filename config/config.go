package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/fsutil"
)

var FS fsutil.FileSystem = fsutil.NewFileSystem()

type CredentialsConfig struct {
	Default Credentials `toml:"default"`
}

type Credentials struct {
	HyphenAccessToken  string `toml:"hyphen_access_token"`
	HyphenRefreshToken string `toml:"hyphen_refresh_token"`
	OrganizationId     string `toml:"organization_id"`
	HypenIDToken       string `toml:"hyphen_id_token"`
	ExpiryTime         int64  `toml:"expiry_time"`
}

// Filepaths for credential storage
const (
	WindowsConfigPath = "Hyphen"
	UnixConfigPath    = ".hyphen"
	CredentialFile    = "credentials.toml"
)

func Init(fs fsutil.FileSystem) {
	FS = fs
}

func getConfigDirectory() string {
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
func SaveCredentials(organizationID, accessToken, refreshToken, IDToken string, expiryTime int64) error {
	configDir := getConfigDirectory()
	if err := ensureDir(configDir); err != nil {
		return errors.Wrap(err, "Failed to create configuration directory")
	}

	credentialFilePath := filepath.Join(configDir, CredentialFile)
	credentialsContent := fmt.Sprintf(
		"[default]\nhyphen_access_token=\"%s\"\nhyphen_refresh_token=\"%s\"\norganization_id=\"%s\"\nhyphen_id_token=\"%s\"\nexpiry_time=%d",
		accessToken, refreshToken, organizationID, IDToken, expiryTime)

	// Write the credentials to the file
	if err := FS.WriteFile(credentialFilePath, []byte(credentialsContent), 0644); err != nil {
		return errors.Wrap(err, "Failed to save credentials")
	}

	return nil
}

// LoadCredentials retrieves credentials from a configuration file
func LoadCredentials() (CredentialsConfig, error) {
	configDir := getConfigDirectory()
	credentialFilePath := filepath.Join(configDir, CredentialFile)

	data, err := FS.ReadFile(credentialFilePath)
	if err != nil {
		return CredentialsConfig{}, fmt.Errorf("Failed to read credentials file: %w", err)
	}

	var config CredentialsConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return CredentialsConfig{}, fmt.Errorf("Failed to parse credentials file: %w", err)
	}

	return config, nil
}

// UpdateOrganizationID updates the organization ID in the credentials file
func UpdateOrganizationID(organizationID string) error {
	credentials, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("Failed to load existing credentials: %w", err)
	}

	credentials.Default.OrganizationId = organizationID

	configDir := getConfigDirectory()
	credentialFilePath := filepath.Join(configDir, CredentialFile)

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(credentials); err != nil {
		return fmt.Errorf("Failed to encode updated credentials: %w", err)
	}

	if err := FS.WriteFile(credentialFilePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("Failed to write updated credentials: %w", err)
	}

	return nil
}
