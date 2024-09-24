package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/stretchr/testify/assert"
)

// TestSaveCredentialsSuccess to test successful credentials saving
func TestSaveCredentialsSuccess(t *testing.T) {
	mockFS := &fsutil.MockFileSystem{}
	FS = mockFS

	credentialFile := filepath.Join(GetConfigDirectory(), CredentialFile)
	expectedCredentials := CredentialsConfig{
		Default: Credentials{
			HyphenAccessToken:  "user_access_token",
			HyphenRefreshToken: "user_refresh_token",
			HypenIDToken:       "user_id_token",
			ExpiryTime:         12,
		},
	}

	mockFS.StatFunc = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	mockFS.MkdirAllFunc = func(path string, perm os.FileMode) error {
		return nil
	}
	mockFS.WriteFileFunc = func(filename string, data []byte, perm os.FileMode) error {
		assert.Equal(t, credentialFile, filename)
		var writtenCredentials CredentialsConfig
		err := json.Unmarshal(data, &writtenCredentials)
		assert.Nil(t, err, "Should be able to unmarshal written data")
		assert.Equal(t, expectedCredentials, writtenCredentials)
		assert.Equal(t, os.FileMode(0644), perm)
		return nil
	}

	err := SaveCredentials("user_access_token", "user_refresh_token", "user_id_token", 12)
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
			err := SaveCredentials("user_access_token", "user_refresh_token", "user_id_token", 12)
			assert.NotNil(t, err, "SaveCredentials should return an error")
		})
	}
}

// TestLoadCredentialsSuccess tests successful credentials retrieval
func TestLoadCredentialsSuccess(t *testing.T) {
	mockFS := &fsutil.MockFileSystem{}
	FS = mockFS

	credentialFile := filepath.Join(GetConfigDirectory(), CredentialFile)
	expectedCredentials := CredentialsConfig{
		Default: Credentials{
			HyphenAccessToken:  "user_access_token",
			HyphenRefreshToken: "user_refresh_token",
			HypenIDToken:       "user_id_token",
			ExpiryTime:         12,
		},
	}

	credentialsJSON, _ := json.Marshal(expectedCredentials)

	mockFS.ReadFileFunc = func(filename string) ([]byte, error) {
		assert.Equal(t, credentialFile, filename)
		return credentialsJSON, nil
	}

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
					return []byte("invalid json data"), nil
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
