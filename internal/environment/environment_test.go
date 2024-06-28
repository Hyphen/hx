package environment

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/secretkey"
)

// Mock SecretKeyer for testing
type mockSecretKey struct{}

func (m *mockSecretKey) Base64() string {
	return "mockBase64Key"
}

func (m *mockSecretKey) HashSHA() string {
	return "mockHash"
}

func (m *mockSecretKey) Encrypt(message string) (string, error) {
	encoded := base64.URLEncoding.EncodeToString([]byte(message))
	return encoded, nil
}

func (m *mockSecretKey) Decrypt(encryptedMessage string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encryptedMessage)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// Mock EnviromentHandler
type MockEnvHandler struct {
	envVars   string
	secretKey secretkey.SecretKeyer
}

func (m *MockEnvHandler) EncryptEnvironmentVars(file string) (string, error) {
	return m.secretKey.Encrypt(m.envVars)
}

func (m *MockEnvHandler) DecryptEnviromentVars(env string) ([]string, error) {
	decrypted, err := m.secretKey.Decrypt(m.envVars)
	if err != nil {
		return nil, err
	}
	return strings.Split(decrypted, "\n"), nil
}

func (m *MockEnvHandler) DecryptedEnviromentVarsIntoAFile(env string) (string, error) {
	decrypted, err := m.DecryptEnviromentVars(env)
	if err != nil {
		return "", err
	}
	tmpFile, tmpFileLocation := tmpFile()
	defer tmpFile.Close()

	for _, envVar := range decrypted {
		_, err := tmpFile.WriteString(envVar + "\n")
		if err != nil {
			return "", fmt.Errorf("error writing environment variables to temporary file: %w", err)
		}
	}

	return tmpFileLocation, nil
}

func (m *MockEnvHandler) GetEncryptedEnviromentVars(env string) (string, error) {
	return m.envVars, nil
}

func (m *MockEnvHandler) UploadEncryptedEnviromentVars(env string) error {
	return nil
}

func (m *MockEnvHandler) SourceEnviromentVars(env string) error {
	return nil
}

func (m *MockEnvHandler) SecretKey() secretkey.SecretKeyer {
	return m.secretKey
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
secret_key = "c2VjcmV0LWtleQ=="
`

	if _, err := file.WriteString(configContent); err != nil {
		t.Fatalf("Unable to write to temp file: %v", err)
	}

	file.Close()

	// Run the RestoreFromFile function
	envHandler := RestoreFromFile(file.Name())

	if envHandler.SecretKey().Base64() != "c2VjcmV0LWtleQ==" {
		t.Errorf("Unexpected SecretKey.Base64() = %v", envHandler.SecretKey().Base64())
	}
}

func TestInitialize(t *testing.T) {
	// Initialize
	env := Initialize("test-app", "test-app")
	if env.SecretKey().Base64() == "" {
		t.Errorf("Expected a new secret key")
	}

	// Check config file
	fileContent, err := os.ReadFile(EnvConfigFile)
	if err != nil {
		t.Fatalf("Unable to read config file: %v", err)
	}

	expectedContent := "app_name = \"test-app\""
	if !strings.Contains(string(fileContent), expectedContent) {
		t.Errorf("Config file does not contain expected content")
	}

	os.Remove(EnvConfigFile) // Clean up
}

func TestEncryptEnvironmentVars(t *testing.T) {
	mockKey := &mockSecretKey{}
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
	mockKey := &mockSecretKey{}
	envHandler := &MockEnvHandler{
		envVars:   base64.URLEncoding.EncodeToString([]byte("TEST_VAR=test_value\nANOTHER_VAR=another_value")),
		secretKey: mockKey,
	}

	// Test DecryptedEnviromentVarsIntoAFile
	tmpFileLocation, err := envHandler.DecryptedEnviromentVarsIntoAFile("mockEnv")
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
	mockKey := &mockSecretKey{}
	envHandler := &MockEnvHandler{
		envVars:   base64.URLEncoding.EncodeToString([]byte("TEST_VAR=test_value\nANOTHER_VAR=another_value")),
		secretKey: mockKey,
	}

	// Test DecryptEnviromentVars
	decryptedVars, err := envHandler.DecryptEnviromentVars("mockEnv")
	if err != nil {
		t.Fatalf("DecryptEnviromentVars error = %v", err)
	}

	expectedDecryptedVars := []string{"TEST_VAR=test_value", "ANOTHER_VAR=another_value"}
	for i, v := range decryptedVars {
		if v != expectedDecryptedVars[i] {
			t.Errorf("Decrypted variable = %v, want %v", v, expectedDecryptedVars[i])
		}
	}
}
