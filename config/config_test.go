package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/stretchr/testify/assert"
)

// TestSaveCredentialsSuccess to test successful credentials saving
func TestSaveCredentialsSuccess(t *testing.T) {
	mockFS := &fsutil.MockFileSystem{}
	FS = mockFS

	credentialFile := filepath.Join(getConfigDirectory(), CredentialFile)
	credentialsContent := fmt.Sprintf(
		"[default]\nhyphen_access_token=\"%s\"\nhyphen_refresh_token=\"%s\"\norganization_id=\"%s\"\nhyphen_id_token=\"%s\"\nexpiry_time=%d",
		"user_access_token", "user_refresh_token", "org_id", "user_id_token", 12)

	mockFS.StatFunc = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
		return nil
	}
	mockFS.WriteFileFunc = func(filename string, data []byte, perm os.FileMode) error {
		assert.Equal(t, credentialFile, filename)
		assert.Equal(t, []byte(credentialsContent), data)
		assert.Equal(t, os.FileMode(0644), perm)
		return nil
	}

	err := SaveCredentials("org_id", "user_access_token", "user_refresh_token", "user_id_token", 12)
	assert.Nil(t, err, "SaveCredentials should not return an error")
}

// TestSaveCredentialsFailures to handle different types of failures
func TestSaveCredentialsFailures(t *testing.T) {
	mockFS := &fsutil.MockFileSystem{}
	FS = mockFS

	tests := []struct {
		name       string
		setupMocks func()
	}{
		{
			name: "failure in directory creation",
			setupMocks: func() {
				mockFS.StatFunc = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
				mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
					return fmt.Errorf("failed to create directory")
				}
			},
		},
		{
			name: "failure in file write",
			setupMocks: func() {
				mockFS.StatFunc = func(name string) (os.FileInfo, error) {
					return &fsutil.MockFileInfo{}, nil
				}
				mockFS.WriteFileFunc = func(filename string, data []byte, perm os.FileMode) error {
					return fmt.Errorf("failed to write file")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()
			err := SaveCredentials("org_id", "user_access_token", "user_refresh_token", "user_id_token", 12)
			assert.NotNil(t, err, "SaveCredentials should return an error")
		})
	}
}

// TestLoadCredentialsSuccess tests successful credentials retrieval
func TestLoadCredentialsSuccess(t *testing.T) {
	mockFS := &fsutil.MockFileSystem{}
	FS = mockFS

	credentialFile := filepath.Join(getConfigDirectory(), CredentialFile)
	credentialsToml := `[default]
hyphen_access_token="user_access_token"
hyphen_refresh_token="user_refresh_token"
organization_id="org_id"
hyphen_id_token="user_id_token"
expiry_time=12`

	mockFS.ReadFileFunc = func(filename string) ([]byte, error) {
		assert.Equal(t, credentialFile, filename)
		return []byte(credentialsToml), nil
	}

	var expectedCredentials CredentialsConfig
	err := toml.Unmarshal([]byte(credentialsToml), &expectedCredentials)
	assert.Nil(t, err, "Unmarshalling should succeed in test setup")

	retrievedCredentials, err := LoadCredentials()
	assert.Nil(t, err, "LoadCredentials should not return an error")
	assert.Equal(t, expectedCredentials, retrievedCredentials, "Retrieved credentials should match expected")
}

// TestLoadCredentialsFailures tests different failure scenarios for credentials retrieval
func TestLoadCredentialsFailures(t *testing.T) {
	mockFS := &fsutil.MockFileSystem{}
	FS = mockFS

	tests := []struct {
		name        string
		setupMocks  func()
		expectedErr string
	}{
		{
			name: "failure reading file",
			setupMocks: func() {
				mockFS.ReadFileFunc = func(filename string) ([]byte, error) {
					return nil, errors.New("failed to read file")
				}
			},
			expectedErr: "Failed to read credentials file: failed to read file",
		},
		{
			name: "failure decoding credentials",
			setupMocks: func() {
				mockFS.ReadFileFunc = func(filename string) ([]byte, error) {
					return []byte("invalid toml data"), nil
				}
			},
			expectedErr: "Failed to parse credentials file: ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()
			_, err := LoadCredentials()
			assert.Contains(t, err.Error(), tc.expectedErr, "Error message should contain expected string for "+tc.name)
		})
	}
}

// TestUpdateOrganizationID tests the UpdateOrganizationID function
func TestUpdateOrganizationID(t *testing.T) {
	mockFS := &fsutil.MockFileSystem{}
	FS = mockFS

	credentialFile := filepath.Join(getConfigDirectory(), CredentialFile)
	initialToml := `[default]
hyphen_access_token="user_access_token"
hyphen_refresh_token="user_refresh_token"
organization_id="old_org_id"
hyphen_id_token="user_id_token"
expiry_time=12`

	mockFS.ReadFileFunc = func(filename string) ([]byte, error) {
		assert.Equal(t, credentialFile, filename)
		return []byte(initialToml), nil
	}

	var writtenData []byte
	mockFS.WriteFileFunc = func(filename string, data []byte, perm os.FileMode) error {
		assert.Equal(t, credentialFile, filename)
		assert.Equal(t, os.FileMode(0644), perm)
		writtenData = data
		return nil
	}

	err := UpdateOrganizationID("new_org_id")
	assert.Nil(t, err, "UpdateOrganizationID should not return an error")

	// Verify that the new organization ID was written
	var updatedConfig CredentialsConfig
	err = toml.Unmarshal(writtenData, &updatedConfig)
	assert.Nil(t, err, "Should be able to unmarshal written data")
	assert.Equal(t, "new_org_id", updatedConfig.Default.OrganizationId, "Organization ID should be updated")
}

func TestUpdateOrganizationIDFailure(t *testing.T) {
	mockFS := &fsutil.MockFileSystem{}
	FS = mockFS

	mockFS.ReadFileFunc = func(filename string) ([]byte, error) {
		return nil, errors.New("failed to read file")
	}

	err := UpdateOrganizationID("new_org_id")
	assert.NotNil(t, err, "UpdateOrganizationID should return an error when reading fails")
}
