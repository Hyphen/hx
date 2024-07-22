package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test for GetConfigDirectory
func TestGetConfigDirectory(t *testing.T) {
	se := systemEnvironment{}
	expectedPath := ""
	if runtime.GOOS == "windows" {
		expectedPath = filepath.Join(os.Getenv("APPDATA"), WindowsConfigPath)
	} else {
		home, _ := os.UserHomeDir() // We ignore error for simplicity in the test case
		expectedPath = filepath.Join(home, UnixConfigPath)
	}
	assert.Equal(t, expectedPath, se.GetConfigDirectory(), "Config directory should match expected path")
}

// Test for EnsureDir
func TestEnsureDir(t *testing.T) {
	dirName := "testdir"
	se := &systemEnvironment{}
	err := se.EnsureDir(dirName)
	assert.Nil(t, err, "EnsureDir should not return an error")

	// Clean up by removing the directory
	cleanupErr := os.Remove(dirName)
	assert.Nil(t, cleanupErr, "Removing test directory failed which can affect other tests")
}

// Test for WriteFile
func TestWriteFile(t *testing.T) {
	se := &systemEnvironment{}
	filename := "testfile.txt"

	// Write to the file
	err := se.WriteFile(filename, []byte("test data"), 0644)
	// Check that the write operation did not return an error
	assert.Nil(t, err, "WriteFile should not return an error")

	// Clean up by removing the file
	cleanupErr := os.Remove(filename)
	assert.Nil(t, cleanupErr, "Removing test file failed which can affect other tests")
}

// Test for ReadFile
func TestReadFile(t *testing.T) {
	filename := "testfile.txt"
	se := &systemEnvironment{}

	err := se.WriteFile(filename, []byte("test data"), 0644)
	assert.Nil(t, err, "WriteFile should not return an error")
	data, err := se.ReadFile("testfile.txt")
	assert.Nil(t, err, "ReadFile should not return an error when reading existing file")
	assert.NotEmpty(t, data, "ReadFile should return data")

	// Clean up by removing the file
	cleanupErr := os.Remove(filename)
	assert.Nil(t, cleanupErr, "Removing test file failed which can affect other tests")
}

// Mock to simulate methods in the Environment interface
type MockEnvironment struct {
	mock.Mock
}

func (m *MockEnvironment) GetConfigDirectory() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEnvironment) EnsureDir(dirName string) error {
	args := m.Called(dirName)
	return args.Error(0)
}

func (m *MockEnvironment) WriteFile(filename string, data []byte, perm os.FileMode) error {
	args := m.Called(filename, data, perm)
	return args.Error(0)
}

func (m *MockEnvironment) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockEnvironment) GetGOOS() string {
	args := m.Called()
	return args.String(0)
}

// TestSaveCredentialsSuccess to test successful credentials saving
func TestSaveCredentialsSuccess(t *testing.T) {
	mockEnv := new(MockEnvironment)
	Env = mockEnv

	configDir := "/mocked/path"
	credentialFile := filepath.Join(configDir, CredentialFile)
	credentialsContent := fmt.Sprintf(
		"[default]\nhyphen_access_token=\"%s\"\nhyphen_refresh_token=\"%s\"\nhyphen_id_token=\"%s\"\nexpiry_time=%d",
		"user_access_token", "user_refresh_token", "user_id_token", 12)

	mockEnv.On("GetConfigDirectory").Return(configDir)
	mockEnv.On("EnsureDir", configDir).Return(nil)
	mockEnv.On("WriteFile", credentialFile, []byte(credentialsContent), os.FileMode(0644)).Return(nil)

	err := SaveCredentials("user_access_token", "user_refresh_token", "user_id_token", 12)
	assert.Nil(t, err, "SaveCredentials should not return an error")

	mockEnv.AssertExpectations(t) // verify that all mocked methods were called as expected
}

// TestSaveCredentialsFailures to handle different types of failures
func TestSaveCredentialsFailures(t *testing.T) {
	mockEnv := new(MockEnvironment)
	Env = mockEnv

	configDir := "/mocked/path"
	credentialFile := filepath.Join(configDir, CredentialFile)
	credentialsContent := fmt.Sprintf(
		"[default]\nhyphen_access_token=\"%s\"\nhyphen_refresh_token=\"%s\"\nhyphen_id_token=\"%s\"\nexpiry_time=%d",
		"user_access_token", "user_refresh_token", "user_id_token", 12)

	tests := []struct {
		name       string
		setupMocks func()
	}{
		{
			name: "failure in directory creation",
			setupMocks: func() {
				mockEnv.On("GetConfigDirectory").Return(configDir)
				mockEnv.On("EnsureDir", configDir).Return(fmt.Errorf("failed to create directory"))
			},
		},
		{
			name: "failure in file write",
			setupMocks: func() {
				mockEnv.On("GetConfigDirectory").Return(configDir)
				mockEnv.On("EnsureDir", configDir).Return(nil)
				mockEnv.On("WriteFile", credentialFile, []byte(credentialsContent), os.FileMode(0644)).Return(fmt.Errorf("failed to write file"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockEnv.ExpectedCalls = nil // reset expectations
			mockEnv.Calls = nil         // clean the call stack
			tc.setupMocks()             // setup test-specific mocks
			err := SaveCredentials("user_access_token", "user_refresh_token", "user_id_token", 12)
			assert.NotNil(t, err, "SaveCredentials should return an error")
			mockEnv.AssertExpectations(t)
		})
	}
}

// TestGetCredentialsSuccess tests successful credentials retrieval
func TestGetCredentialsSuccess(t *testing.T) {
	mockEnv := new(MockEnvironment)
	Env = mockEnv

	configDir := "/mocked/path"
	credentialFile := filepath.Join(configDir, CredentialFile)
	credentialsToml := `[default]
hyphen_access_token="user_access_token"
hyphen_refresh_token="user_refresh_token"
hyphen_id_token="user_id_token"
expiry_time=12`

	mockEnv.On("GetConfigDirectory").Return(configDir)
	mockEnv.On("ReadFile", credentialFile).Return([]byte(credentialsToml), nil)

	var credentials CredentialsConfig
	err := toml.Unmarshal([]byte(credentialsToml), &credentials)
	assert.Nil(t, err, "Unmarshalling should succeed in test setup")

	retrievedCredentials, err := LoadCredentials()
	assert.Nil(t, err, "GetCredentials should not return an error")
	assert.Equal(t, credentials, retrievedCredentials, "Retrieved credentials should match expected")

	mockEnv.AssertExpectations(t) // Verify that all mocked methods were called as expected
}

// TestGetCredentialsFailures tests different failure scenarios for credentials retrieval
func TestGetCredentialsFailures(t *testing.T) {
	mockEnv := new(MockEnvironment)
	Env = mockEnv

	configDir := "/mocked/path"
	credentialFile := filepath.Join(configDir, CredentialFile)
	mockEnv.On("GetConfigDirectory").Return(configDir)

	tests := []struct {
		name        string
		setupMocks  func()
		expectedErr string
	}{
		{
			name: "failure reading file",
			setupMocks: func() {
				mockEnv.On("ReadFile", credentialFile).Return([]byte(nil), errors.New("failed to read file")).Once()
			},
			expectedErr: "failed to read file: failed to read file",
		},

		{
			name: "failure decoding credentials",
			setupMocks: func() {
				// Return some bytes to simulate bad TOML data
				mockEnv.On("ReadFile", credentialFile).Return([]byte("invalid toml data"), nil).Once()
			},
			expectedErr: "failed to decode credentials: toml: line 1: expected '.' or '=', but got 't' instead",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()
			_, err := LoadCredentials()
			assert.EqualError(t, err, tc.expectedErr, "Error message should match for "+tc.name)
			mockEnv.AssertExpectations(t)
		})
	}
}
