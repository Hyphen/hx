package environment

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/environment/envvars"
	"github.com/stretchr/testify/assert"
)

// MockRepository implements the Repository interface for testing
type MockRepository struct {
	InitializeFunc            func(apiName, apiId string) error
	GetEncryptedVariablesFunc func(env, appID string) (string, error)
	UploadEnvVariableFunc     func(env, appID string, envData envvars.EnviromentVarsData) error
}

func (m *MockRepository) Initialize(apiName, apiId string) error {
	if m.InitializeFunc != nil {
		return m.InitializeFunc(apiName, apiId)
	}
	return nil
}

func (m *MockRepository) GetEncryptedVariables(env, appID string) (string, error) {
	if m.GetEncryptedVariablesFunc != nil {
		return m.GetEncryptedVariablesFunc(env, appID)
	}
	return "", nil
}

func (m *MockRepository) UploadEnvVariable(env, appID string, envData envvars.EnviromentVarsData) error {
	if m.UploadEnvVariableFunc != nil {
		return m.UploadEnvVariableFunc(env, appID, envData)
	}
	return nil
}

// MockSecretKeyer for testing
type MockSecretKey struct{}

func (m *MockSecretKey) Base64() string {
	return "mockBase64Key"
}

func (m *MockSecretKey) HashSHA() string {
	return "mockHash"
}

func (m *MockSecretKey) Encrypt(message string) (string, error) {
	encoded := base64.URLEncoding.EncodeToString([]byte(message))
	return encoded, nil
}

func (m *MockSecretKey) Decrypt(encryptedMessage string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encryptedMessage)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func TestRestoreFromFile(t *testing.T) {
	// Create a temporary config file
	file, err := os.CreateTemp("", "env-config")
	if err != nil {
		t.Fatalf("Unable to create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	configContent := `
app_name = "test-app"
app_id = "test-app"
secret_key = "mockBase64Key"
`

	if _, err := file.WriteString(configContent); err != nil {
		t.Fatalf("Unable to write to temp file: %v", err)
	}

	file.Close()

	// Run the RestoreFromFile function
	SetRepository(&MockRepository{}) // Use mock repository
	envHandler := RestoreFromFile(file.Name())

	if envHandler.SecretKey().Base64() != "mockBase64Key" {
		t.Errorf("Unexpected SecretKey.Base64() = %v", envHandler.SecretKey().Base64())
	}
}

func TestInitialize(t *testing.T) {
	mockRepo := &MockRepository{
		InitializeFunc: func(apiName, apiId string) error {
			return nil
		},
	}
	SetRepository(mockRepo)

	// Initialize environment
	env := Initialize("test-app", "test-app")
	if env.SecretKey().Base64() == "" {
		t.Errorf("Expected a new secret key")
	}

	// Check config file
	fileContent, err := os.ReadFile(EnvConfigFile)
	if err != nil {
		t.Fatalf("Unable to read config file: %v", err)
	}

	expectedContent := `app_name = "test-app"`
	if !strings.Contains(string(fileContent), expectedContent) {
		t.Errorf("Config file does not contain expected content")
	}

	os.Remove(EnvConfigFile) // Clean up
}

func TestEncryptEnvironmentVars(t *testing.T) {
	mockKey := &MockSecretKey{}
	env := &Enviroment{secretKey: mockKey}

	// Create a temporary file
	file, err := os.CreateTemp("", "env-vars")
	if err != nil {
		t.Fatalf("Unable to create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	fileContent := "TEST_VAR=test_value\nANOTHER_VAR=another_value"
	if _, err := file.WriteString(fileContent); err != nil {
		t.Fatalf("Unable to write to temp file: %v", err)
	}
	file.Close()

	// Test EncryptEnvironmentVars
	encrypted, err := env.EncryptEnvironmentVars(file.Name())
	if err != nil {
		t.Fatalf("EncryptEnvironmentVars error = %v", err)
	}

	expectedEncrypted := base64.URLEncoding.EncodeToString([]byte(fileContent))
	if encrypted != expectedEncrypted {
		t.Errorf("EncryptEnvironmentVars = %v, want %v", encrypted, expectedEncrypted)
	}
}

func TestDecryptEnvironmentVarsIntoAFile(t *testing.T) {
	mockKey := &MockSecretKey{}
	mockRepo := &MockRepository{}
	SetRepository(mockRepo)

	env := &Enviroment{
		secretKey:  mockKey,
		repository: mockRepo,
	}
	envVars := base64.URLEncoding.EncodeToString([]byte("TEST_VAR=test_value\nANOTHER_VAR=another_value"))

	// Mock GetEncryptedVariables to return envVars
	mockRepo.GetEncryptedVariablesFunc = func(env, appID string) (string, error) {
		return envVars, nil
	}

	tmpFileLocation, err := env.DecryptedEnviromentVarsIntoAFile("mockEnv", "mockApp")
	if err != nil {
		t.Fatalf("DecryptedEnviromentVarsIntoAFile error = %v", err)
	}
	defer os.Remove(tmpFileLocation)

	fileContent, err := os.ReadFile(tmpFileLocation)
	if err != nil {
		t.Fatalf("Unable to read decrypted file: %v", err)
	}

	expectedContent := "TEST_VAR=test_value\nANOTHER_VAR=another_value\n"
	if string(fileContent) != expectedContent {
		t.Errorf("Decrypted file content = %v, want %v", string(fileContent), expectedContent)
	}
}

func TestEncryptAndDecryptEnvironmentVars(t *testing.T) {
	mockKey := &MockSecretKey{}
	env := &Enviroment{secretKey: mockKey}

	// Create a temporary file
	file, err := os.CreateTemp("", "env-vars")
	if err != nil {
		t.Fatalf("Unable to create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	fileContent := "TEST_VAR=test_value\nANOTHER_VAR=another_value"
	if _, err := file.WriteString(fileContent); err != nil {
		t.Fatalf("Unable to write to temp file: %v", err)
	}
	file.Close()

	// Test EncryptEnvironmentVars
	encrypted, err := env.EncryptEnvironmentVars(file.Name())
	if err != nil {
		t.Fatalf("EncryptEnvironmentVars error = %v", err)
	}

	// Test DecryptEnvironmentVars
	decryptedVars, err := env.secretKey.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("DecryptEnvironmentVars error = %v", err)
	}

	expectedDecryptedVars := strings.Split(fileContent, "\n")
	actualDecryptedVars := strings.Split(decryptedVars, "\n")
	for i, v := range actualDecryptedVars {
		if v != expectedDecryptedVars[i] {
			t.Errorf("Decrypted variable = %v, want %v", v, expectedDecryptedVars[i])
		}
	}
}

func TestGetEnvName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"Empty input", "", "default", false},
		{"Lowercase input", "production", "production", false},
		{"Uppercase input", "STAGING", "staging", false},
		{"Mixed case input", "Dev", "dev", false},
		{"Invalid characters", "Test 123!@#", "", true},
		{"Valid with hyphen and underscore", "test-env_1", "test-env_1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetEnvName(tt.input)

			if tt.hasError {
				assert.Error(t, err, "Expected an error for input: %s", tt.input)
				assert.Empty(t, result, "Expected empty result for input: %s", tt.input)
			} else {
				assert.NoError(t, err, "Unexpected error for input: %s", tt.input)
				assert.Equal(t, tt.expected, result, "Unexpected result for input: %s", tt.input)
			}
		})
	}
}

func TestGetEnvFileByEnvironment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ".env"},
		{"default", ".env"},
		{"production", ".env.production"},
		{"STAGING", ".env.staging"},
		{"Dev", ".env.dev"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GetEnvFileByEnvironment(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUploadEncryptedEnviromentVars(t *testing.T) {
	mockKey := &MockSecretKey{}
	mockRepo := &MockRepository{}
	env := &Enviroment{
		secretKey:  mockKey,
		repository: mockRepo,
		config:     Config{AppId: "test-app-id"},
	}

	mockRepo.UploadEnvVariableFunc = func(env, appID string, envData envvars.EnviromentVarsData) error {
		assert.Equal(t, "test-env", env)
		assert.Equal(t, "test-app-id", appID)
		assert.NotEmpty(t, envData.Data)
		return nil
	}

	envData := envvars.EnviromentVarsData{
		Size:           "10",
		CountVariables: 2,
		Data:           "TEST_VAR=test_value\nANOTHER_VAR=another_value",
	}

	err := env.UploadEncryptedEnviromentVars("test-env", envData)
	assert.NoError(t, err)
}

func TestGetEncryptedEnviromentVars(t *testing.T) {
	mockRepo := &MockRepository{}
	env := &Enviroment{
		repository: mockRepo,
		config:     Config{AppId: "test-app-id"},
	}

	expectedVars := "encrypted_vars"
	mockRepo.GetEncryptedVariablesFunc = func(env, appID string) (string, error) {
		assert.Equal(t, "test-env", env)
		assert.Equal(t, "test-app-id", appID)
		return expectedVars, nil
	}

	vars, err := env.GetEncryptedEnviromentVars("test-env")
	assert.NoError(t, err)
	assert.Equal(t, expectedVars, vars)
}

func TestEnsureGitignore(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "gitignore-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Change to the temporary directory
	oldWd, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer os.Chdir(oldWd)

	// Create a .git directory to simulate a Git project
	err = os.Mkdir(".git", 0755)
	assert.NoError(t, err)

	// Test when .gitignore doesn't exist
	err = EnsureGitignore()
	assert.NoError(t, err)

	content, err := os.ReadFile(".gitignore")
	assert.NoError(t, err)
	assert.Contains(t, string(content), EnvConfigFile)

	// Test when .gitignore already exists
	err = os.WriteFile(".gitignore", []byte("existing_entry\n"), 0644)
	assert.NoError(t, err)

	err = EnsureGitignore()
	assert.NoError(t, err)

	content, err = os.ReadFile(".gitignore")
	assert.NoError(t, err)
	assert.Contains(t, string(content), "existing_entry")
	assert.Contains(t, string(content), EnvConfigFile)
}

func TestConfigExists(t *testing.T) {
	// Test when config doesn't exist
	assert.False(t, ConfigExists())

	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "env-config")
	assert.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Set the EnvConfigFile to the temporary file
	oldConfigFile := EnvConfigFile
	EnvConfigFile = tmpFile.Name()
	defer func() { EnvConfigFile = oldConfigFile }()

	// Test when config exists
	assert.True(t, ConfigExists())
}

func TestDecryptEnvironmentVars(t *testing.T) {
	mockKey := &MockSecretKey{}
	mockRepo := &MockRepository{}
	env := &Enviroment{
		secretKey:  mockKey,
		repository: mockRepo,
		config:     Config{AppId: "test-app-id"},
	}

	encryptedVars := base64.URLEncoding.EncodeToString([]byte("TEST_VAR=test_value\nANOTHER_VAR=another_value"))
	mockRepo.GetEncryptedVariablesFunc = func(env, appID string) (string, error) {
		return encryptedVars, nil
	}

	vars, err := env.DecryptEnvironmentVars("test-env")
	assert.NoError(t, err)
	assert.Equal(t, []string{"TEST_VAR=test_value", "ANOTHER_VAR=another_value"}, vars)
}

func TestSetRepository(t *testing.T) {
	originalRepo := repository
	defer func() { repository = originalRepo }()

	newRepo := &MockRepository{}
	SetRepository(newRepo)

	assert.Equal(t, newRepo, repository)
}

func TestRestore(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "env-config")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	configContent := `
app_name = "test-app"
app_id = "test-app-id"
secret_key = "mockBase64Key"
`
	_, err = tmpFile.WriteString(configContent)
	assert.NoError(t, err)
	tmpFile.Close()

	// Set the EnvConfigFile to the temporary file
	oldConfigFile := EnvConfigFile
	EnvConfigFile = tmpFile.Name()
	defer func() { EnvConfigFile = oldConfigFile }()

	// Run the Restore function
	SetRepository(&MockRepository{})
	envHandler := Restore()

	assert.NotNil(t, envHandler)
	assert.Equal(t, "mockBase64Key", envHandler.SecretKey().Base64())
}

func TestTmpDir(t *testing.T) {
	dir := tmpDir()
	defer os.RemoveAll(dir)

	assert.DirExists(t, dir)
}

func TestTmpFile(t *testing.T) {
	file, name := tmpFile()
	defer os.Remove(name)

	assert.NotNil(t, file)
	assert.FileExists(t, name)
}
