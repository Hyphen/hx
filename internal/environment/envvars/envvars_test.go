package envvars

import (
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/secretkey"
)

func createTestFile(t *testing.T, content string) string {
	file, err := os.CreateTemp("", "envvars_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = file.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	err = file.Close()
	if err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return file.Name()
}

func TestNew(t *testing.T) {
	envContent := "VAR1=value1\nVAR2=value2\nVAR3=value3\n"
	fileName := createTestFile(t, envContent)
	defer os.Remove(fileName)

	data, err := New(fileName)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedVarsCount := 3
	if data.CountVariables != expectedVarsCount {
		t.Errorf("Expected %d variables, got %d", expectedVarsCount, data.CountVariables)
	}

	if data.Size != "36 bytes" {
		t.Errorf("Expected size '36 bytes', got '%s'", data.Size)
	}

	if data.Data != envContent {
		t.Errorf("Expected content '%s', got '%s'", envContent, data.Data)
	}
}

func TestEncryptData(t *testing.T) {
	envContent := "VAR1=value1\nVAR2=value2\nVAR3=value3\n"
	fileName := createTestFile(t, envContent)
	defer os.Remove(fileName)

	data, err := New(fileName)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	key := secretkey.New()
	err = data.EncryptData(key)
	if err != nil {
		t.Fatalf("Unexpected error during encryption: %v", err)
	}

	// Ensure the content is encrypted
	if data.Data == envContent {
		t.Errorf("Data was not encrypted")
	}

	// Decrypt to confirm encryption is reversible
	decryptedData, err := key.Decrypt(data.Data)
	if err != nil {
		t.Fatalf("Unexpected error during decryption: %v", err)
	}

	if decryptedData != envContent {
		t.Errorf("Expected decrypted data '%s', got '%s'", envContent, decryptedData)
	}
}
