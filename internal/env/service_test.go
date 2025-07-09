package env

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNew(t *testing.T) {
	// Create a temporary .env file
	content := "KEY1=VALUE1\nKEY2=VALUE2\n"
	tmpfile, err := os.CreateTemp("", "test.env")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	tmpfile.Close()

	// Test New function
	env, err := New(tmpfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "24 bytes", env.Size)
	assert.Equal(t, 2, env.CountVariables)
	assert.Equal(t, content, env.Data)
	assert.Nil(t, env.Version)
}

func GetBase64Secret() string {
	nonBase64SecretValue := "test-secret-test-secret-test-secret-test-secret-test-secret-test-secret"
	base64SecretValue := base64.StdEncoding.EncodeToString([]byte(nonBase64SecretValue))
	return base64SecretValue
}

func TestEncryptData(t *testing.T) {
	env := models.Env{Data: "KEY=VALUE"}
	secret := models.NewSecret(GetBase64Secret())

	encryptedData, err := env.EncryptData(secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, encryptedData)
	assert.NotEqual(t, "KEY=VALUE", encryptedData) // Should be encrypted
}

func TestDecryptData(t *testing.T) {
	// First encrypt some data, then decrypt it
	secret := models.NewSecret(GetBase64Secret())
	originalData := "KEY=VALUE"

	encryptedData, err := secret.Encrypt(originalData)
	assert.NoError(t, err)

	env := models.Env{Data: encryptedData}
	decryptedData, err := env.DecryptData(secret)
	assert.NoError(t, err)
	assert.Equal(t, originalData, decryptedData)
}

func TestDecryptVarsAndSaveIntoFile(t *testing.T) {
	// Encrypt the test data first
	secret := models.NewSecret(GetBase64Secret())
	originalData := "KEY1=VALUE1\nKEY2=VALUE2"

	encryptedData, err := secret.Encrypt(originalData)
	assert.NoError(t, err)

	env := models.Env{Data: encryptedData}

	tmpfile, err := os.CreateTemp("", "test_decrypted.env")
	assert.NoError(t, err)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	fileName, err := env.DecryptVarsAndSaveIntoFile(tmpfile.Name(), secret)
	assert.NoError(t, err)
	assert.Equal(t, tmpfile.Name(), fileName)

	content, err := os.ReadFile(tmpfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, originalData, string(content))
}

func TestGetEnvName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"", "default", false},
		{"production", "production", false},
		{"STAGING", "staging", false},
		{"dev-env", "dev-env", false},
		{"test_env", "test_env", false},
		{"Invalid Env", "", true},
		{"123-abc_DEF", "123-abc_def", false},
	}

	for _, test := range tests {
		result, err := GetEnvName(test.input)
		if test.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		}
	}
}

func TestGetFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"", ".env", false},
		{"production", ".env.production", false},
		{"STAGING", ".env.staging", false},
		{"dev-env", ".env.dev-env", false},
		{"test_env", ".env.test_env", false},
		{"Invalid Env", "", true},
	}

	for _, test := range tests {
		result, err := GetFileName(test.input)
		if test.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		}
	}
}

func TestEnvService_GetEnvironment(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	service := &EnvService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	expectedEnv := models.Environment{ID: "123", Name: "TestEnv"}
	responseBody, _ := json.Marshal(expectedEnv)

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	env, found, err := service.GetEnvironment("org1", "app1", "env1")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, expectedEnv, env)

	mockHTTPClient.AssertExpectations(t)
}

func TestEnvService_PutEnv(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	service := &EnvService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	env := models.Env{Size: "100 bytes", CountVariables: 5}

	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	err := service.PutEnvironmentEnv("org1", "app1", "env1", 12345, env)
	assert.NoError(t, err)

	mockHTTPClient.AssertExpectations(t)
}

func TestEnvService_GetEnv(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	service := &EnvService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	expectedEnv := models.Env{Size: "100 bytes", CountVariables: 5}
	responseBody, _ := json.Marshal(expectedEnv)

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	var secretKeyId int64 = 123
	env, err := service.GetEnvironmentEnv("org1", "app1", "env1", &secretKeyId, nil)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, env)

	mockHTTPClient.AssertExpectations(t)
}

func TestEnvService_ListEnvs(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	service := &EnvService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	expectedEnvs := []models.Env{
		{Size: "100 bytes", CountVariables: 5},
		{Size: "200 bytes", CountVariables: 10},
	}
	envsData := struct {
		Data       []models.Env `json:"data"`
		TotalCount int          `json:"totalCount"`
		PageNum    int          `json:"pageNum"`
		PageSize   int          `json:"pageSize"`
	}{
		Data:       expectedEnvs,
		TotalCount: 2,
		PageNum:    1,
		PageSize:   10,
	}
	responseBody, _ := json.Marshal(envsData)

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	envs, err := service.ListEnvs("org1", "app1", 10, 1)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnvs, envs)
	assert.Len(t, envs, 2)

	mockHTTPClient.AssertExpectations(t)
}

func TestEnvService_ListEnvironments(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	service := &EnvService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	expectedEnvs := []models.Environment{
		{ID: "env1", Name: "Env 1"},
		{ID: "env2", Name: "Env 2"},
	}
	envsData := struct {
		Data       []models.Environment `json:"data"`
		TotalCount int                  `json:"totalCount"`
		PageNum    int                  `json:"pageNum"`
		PageSize   int                  `json:"pageSize"`
	}{
		Data:       expectedEnvs,
		TotalCount: 2,
		PageNum:    1,
		PageSize:   10,
	}
	responseBody, _ := json.Marshal(envsData)

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	envs, err := service.ListEnvironments("org1", "app1", 10, 1)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnvs, envs)
	assert.Len(t, envs, 2)

	mockHTTPClient.AssertExpectations(t)
}

func TestGetEnvsInDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "env_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to the temporary directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tempDir)
	assert.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create some test files
	testFiles := []string{".env", ".env.local", "config.txt", ".env.production"}
	for _, file := range testFiles {
		_, err := os.Create(file)
		assert.NoError(t, err)
	}

	// Run the function
	envFiles, err := GetEnvsInDirectory()
	assert.NoError(t, err)

	// Check the results
	expectedFiles := []string{".env", ".env.production"}
	assert.ElementsMatch(t, expectedFiles, envFiles)
}
