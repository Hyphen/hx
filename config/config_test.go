package config

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
)

type mockEnvironment struct {
	ConfigDir    string
	Files        map[string][]byte
	EnsureDirErr error
	WriteFileErr error
	ReadFileErr  error
}

func (m *mockEnvironment) ReadFile(path string) ([]byte, error) {
	if m.ReadFileErr != nil {
		return nil, m.ReadFileErr
	}
	data, exists := m.Files[path]
	if !exists {
		return nil, fmt.Errorf("file %s does not exist", path)
	}
	return data, nil
}

func (m *mockEnvironment) GetConfigDirectory() string {
	return m.ConfigDir
}

func (m *mockEnvironment) EnsureDir(dirName string) error {
	return m.EnsureDirErr
}

func (m *mockEnvironment) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if m.WriteFileErr != nil {
		return m.WriteFileErr
	}
	m.Files[filename] = data
	return nil
}

func (m *mockEnvironment) DecodeFile(filePath string, v interface{}) error {
	data, ok := m.Files[filePath]
	if !ok {
		return fmt.Errorf("open %s: no such file or directory", filePath)
	}
	if _, err := toml.Decode(string(data), v); err != nil {
		return err
	}
	return nil
}

// TestSaveCredentials checks the behavior of SaveCredentials.
func TestSaveCredentials(t *testing.T) {
	// Arrange
	mockEnv := &mockEnvironment{
		ConfigDir:    "/fake/dir",
		Files:        make(map[string][]byte),
		EnsureDirErr: nil,
		WriteFileErr: nil,
	}
	Env = mockEnv
	username := "user"
	password := "pass"

	// Act
	SaveCredentials(username, password)

	// Assert
	expectedFileName := "/fake/dir/credentials.toml"
	data, exists := mockEnv.Files[expectedFileName]
	if !exists {
		t.Fatalf("Expected file %s to be written", expectedFileName)
	}

	expectedContent := "[default]\nhyphen_access_token=\"user:pass\""
	if !bytes.Equal(data, []byte(expectedContent)) {
		t.Errorf("Expected file content to be %s, got %s", expectedContent, string(data))
	}
}

func TestSaveCredentials_DirectoryCreationFail(t *testing.T) {
	// Arrange
	mockEnv := &mockEnvironment{
		ConfigDir:    "/fake/dir",
		Files:        make(map[string][]byte),
		EnsureDirErr: fmt.Errorf("mock directory creation error"), // Simulate error
	}
	Env = mockEnv
	username := "user"
	password := "pass"

	// Act
	SaveCredentials(username, password)

	// Assert
	expectedFileName := "/fake/dir/credentials.toml"
	_, exists := mockEnv.Files[expectedFileName]
	if exists {
		t.Fatalf("File %s should not be written due to directory creation failure", expectedFileName)
	}
}

func TestSaveCredentials_WriteFileFail(t *testing.T) {
	// Arrange
	mockEnv := &mockEnvironment{
		ConfigDir:    "/fake/dir",
		Files:        make(map[string][]byte),
		WriteFileErr: fmt.Errorf("mock write file error"), // Simulate error
	}
	Env = mockEnv
	username := "user"
	password := "pass"

	// Act
	SaveCredentials(username, password)

	// Assert
	expectedFileName := "/fake/dir/credentials.toml"
	if _, exists := mockEnv.Files[expectedFileName]; exists {
		t.Fatalf("File %s should not be written due to write failure", expectedFileName)
	}
}

// TestGetCredentials attempts to retrieve the credentials using a mocked file reader.
func TestGetCredentials(t *testing.T) {
	mockFiles := map[string][]byte{
		"/fake/dir/credentials.toml": []byte("[default]\nhyphen_access_token=\"user:pass\""),
	}

	mockEnv := &mockEnvironment{
		ConfigDir: "/fake/dir",
		Files:     mockFiles,
	}
	Env = mockEnv // Set the global environment to our mock

	creds, err := GetCredentials()
	if err != nil {
		t.Fatalf("Expected no error, but got: %s", err)
	}

	expectedAccessToken := "user:pass"
	if creds.Default.HyphenAccessToken != expectedAccessToken {
		t.Errorf("Expected hyphen_access_token to be %s, got %s", expectedAccessToken, creds.Default.HyphenAccessToken)
	}
}

func TestGetCredentials_ReadFileFail(t *testing.T) {
	// Arrange
	mockEnv := &mockEnvironment{
		ConfigDir:   "/fake/dir",
		Files:       make(map[string][]byte),
		ReadFileErr: fmt.Errorf("mock read file error"), // Simulate error
	}
	Env = mockEnv

	// Act
	_, err := GetCredentials()

	// Assert
	if err == nil {
		t.Fatal("Expected an read file error, got nil")
	}
}

func TestGetCredentials_DecodeFail(t *testing.T) {
	// Arrange
	corruptData := []byte("[default]\nhyphen_access_token= broken\"data\"")
	mockEnv := &mockEnvironment{
		ConfigDir: "/fake/dir",
		Files: map[string][]byte{
			"/fake/dir/credentials.toml": corruptData,
		},
	}
	Env = mockEnv

	// Act
	_, err := GetCredentials()

	// Assert
	if err == nil {
		t.Fatal("Expected a TOML decoding error, got nil")
	}
}

func TestEnsureDir_Exists(t *testing.T) {
	// Create a temporary directory and ensure it exists
	tmpDir := t.TempDir()

	env := systemEnvironment{}
	if err := env.EnsureDir(tmpDir); err != nil {
		t.Errorf("Expected no error for existing directory, got: %v", err)
	}
}
