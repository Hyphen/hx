package encrypt

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/environment/envvars"
	"github.com/Hyphen/cli/internal/secretkey"
)

// mockEnvHandler now fields the correct signature for all methods in environment.EnvironmentHandler
type mockEnvHandler struct{}

func (m *mockEnvHandler) EncryptEnvironmentVars(file string) (string, error) {
	return "mock_encrypted_data", nil
}

func (m *mockEnvHandler) DecryptEnvironmentVars(env string) ([]string, error) {
	return []string{"VAR=VALUE"}, nil
}

func (m *mockEnvHandler) DecryptedEnvironmentVarsIntoAFile(env, fileName string) (string, error) {
	return "", nil
}

func (m *mockEnvHandler) GetEncryptedEnvironmentVars(env string) (string, error) {
	return "", nil
}

func (m *mockEnvHandler) UploadEncryptedEnvironmentVars(env string, envData envvars.EnvironmentVarsData) error {
	return nil
}

func (m *mockEnvHandler) SourceEnvironmentVars(env string) error {
	return nil
}

func (m *mockEnvHandler) SecretKey() secretkey.SecretKeyer {
	return nil
}

func (m *mockEnvHandler) ListEnvironments(pageSize, pageNum int) ([]envvars.EnvironmentInformation, error) {
	return []envvars.EnvironmentInformation{
		{
			Size:           "235 bytes",
			CountVariables: 6,
			Data:           "mockData1",
			AppId:          "test",
			EnvId:          "default",
		},
		{
			Size:           "34 bytes",
			CountVariables: 1,
			Data:           "mockData2",
			AppId:          "test",
			EnvId:          "prod",
		},
	}, nil
}

func TestEncryptCmd(t *testing.T) {
	// Setup temporary test file
	file, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Unable to create test file: %v", err)
	}
	defer os.Remove(file.Name()) // Clean up the test file

	// Write some test data to the file
	_, err = file.WriteString("TEST_VAR=VALUE\nANOTHER_VAR=ANOTHER_VALUE")
	if err != nil {
		t.Fatalf("Unable to write to test file: %v", err)
	}
	file.Close()

	// Set the mock environment handler
	setEnvHandler(&mockEnvHandler{})

	// Create a test command
	cmd := EncryptCmd
	cmd.SetArgs([]string{file.Name()})

	// Execute the command
	output := captureOutput(func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}
	})

	expectedOutput := "Encrypted data:\nmock_encrypted_data\n"
	if output != expectedOutput {
		t.Errorf("expected %q, got %q", expectedOutput, output)
	}
}

// Utility function to capture output
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String()
}
