package environment

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/environment/envvars"
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
