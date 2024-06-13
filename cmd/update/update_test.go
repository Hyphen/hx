package update

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	cliVersion "github.com/Hyphen/cli/cmd/version"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type MockHTTPClient struct {
	Response *http.Response
	Err      error
}

func (m MockHTTPClient) Get(url string) (*http.Response, error) {
	return m.Response, m.Err
}

type MockFileHandler struct {
	CreateTempFile *os.File
	CreateTempErr  error
	WriteFileErr   error
	ChmodErr       error
	RenameErr      error
}

func (m MockFileHandler) CreateTemp(dir, pattern string) (*os.File, error) {
	return m.CreateTempFile, m.CreateTempErr
}

func (m MockFileHandler) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return m.WriteFileErr
}

func (m MockFileHandler) Chmod(name string, mode os.FileMode) error {
	return m.ChmodErr
}

func (m MockFileHandler) Rename(oldpath, newpath string) error {
	return m.RenameErr
}

func TestNewDefaultUpdater(t *testing.T) {
	version := "1.0.0"
	updater := NewDefaultUpdater(version)

	expectedBaseURL := "https://api.hyphen.ai"
	envBaseURL := os.Getenv("HYPHEN_ENGINE_BASE_URL")
	if envBaseURL != "" {
		expectedBaseURL = envBaseURL
	}

	expectedURLTemplate := expectedBaseURL + "/api/downloads/hyphen-cli/%s?os=%s"
	assert.Equal(t, version, updater.Version, "The version should be set correctly")
	assert.Equal(t, expectedURLTemplate, updater.URLTemplate, "The URLTemplate should be set correctly")
	assert.IsType(t, DefaultHTTPClient{}, updater.HTTPClient, "The HTTPClient should be of type DefaultHTTPClient")
	assert.IsType(t, DefaultFileHandler{}, updater.FileHandler, "The FileHandler should be of type DefaultFileHandler")
	assert.NotNil(t, updater.GetExecPath, "The GetExecPath function should be set")
	assert.NotNil(t, updater.DetectPlatform, "The DetectPlatform function should be set")
}

func TestNewDefaultUpdater_EnvVar(t *testing.T) {
	version := "1.0.0"
	envBaseURL := "https://custom.hyphen.ai"
	os.Setenv("HYPHEN_ENGINE_BASE_URL", envBaseURL)
	defer os.Unsetenv("HYPHEN_ENGINE_BASE_URL")

	updater := NewDefaultUpdater(version)

	expectedURLTemplate := envBaseURL + "/api/downloads/hyphen-cli/%s?os=%s"
	assert.Equal(t, expectedURLTemplate, updater.URLTemplate, "The URLTemplate should be based on the environment variable")
}

func TestUpdater_Run_AlreadyUpToDate(t *testing.T) {
	latestVersion := "1.0.0"
	cliVersion.Version = latestVersion

	mockHTTPClient := MockHTTPClient{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"version":"1.0.0"}]}`)),
		},
		Err: nil,
	}

	updater := &Updater{
		Version:           "latest",
		BaseURL:           "https://api.hyphen.ai",
		URLTemplate:       "https://api.hyphen.ai/api/downloads/hyphen-cli/%s?os=%s",
		HTTPClient:        mockHTTPClient,
		FileHandler:       DefaultFileHandler{},
		GetExecPath:       func() string { return "" },
		DetectPlatform:    func() string { return linux },
		DownloadAndUpdate: func(url string) error { return nil },
	}

	cmd := &cobra.Command{}
	args := []string{}

	// Capture the standard output
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	updater.Run(cmd, args)

	// Restore the real stdout
	w.Close()
	os.Stdout = old
	var buf strings.Builder
	io.Copy(&buf, r)

	expectedOutput := "You are already using the latest version of Hyphen CLI.\n"

	assert.Contains(t, buf.String(), expectedOutput, "The output should indicate that the version is already up-to-date")
}

func TestUpdater_Run_DownloadAndUpdateError(t *testing.T) {
	mockHTTPClient := MockHTTPClient{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"version":"2.0.0"}]}`)),
		},
		Err: nil,
	}
	mockFileHandler := MockFileHandler{CreateTempFile: nil, CreateTempErr: nil}

	updater := &Updater{
		Version:           "latest",
		BaseURL:           "https://api.hyphen.ai",
		URLTemplate:       "https://api.hyphen.ai/api/downloads/hyphen-cli/%s?os=%s",
		HTTPClient:        mockHTTPClient,
		FileHandler:       mockFileHandler,
		GetExecPath:       func() string { return "" },
		DetectPlatform:    func() string { return linux },
		DownloadAndUpdate: func(url string) error { return errors.New("test error") },
	}

	cmd := &cobra.Command{}
	args := []string{}

	// Capture the standard output
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	updater.Run(cmd, args)

	// Restore the real stdout
	w.Close()
	os.Stdout = old
	var buf strings.Builder
	io.Copy(&buf, r)

	expectedOutput := "Failed to update Hyphen CLI: test error\n"

	assert.Contains(t, buf.String(), expectedOutput, "The output should indicate the failure to update with the error message")
}

func TestUpdater_Run_Success(t *testing.T) {
	cliVersion.Version = "1.0.0" // Current version is different from the latest

	mockResp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"version":"2.0.0"}]}`))}
	mockHTTPClient := MockHTTPClient{Response: mockResp, Err: nil}
	mockFile, _ := os.CreateTemp("", "hyphen")
	mockFileHandler := MockFileHandler{CreateTempFile: mockFile, CreateTempErr: nil}

	defer os.Remove(mockFile.Name())

	updater := Updater{
		Version:           "latest",
		BaseURL:           "https://api.hyphen.ai",
		URLTemplate:       "https://api.hyphen.ai/api/downloads/hyphen-cli/%s?os=%s",
		HTTPClient:        mockHTTPClient,
		FileHandler:       mockFileHandler,
		GetExecPath:       func() string { return mockFile.Name() },
		DetectPlatform:    func() string { return linux },
		DownloadAndUpdate: nil,
	}

	updater.DownloadAndUpdate = updater.downloadAndUpdate // Initialize the function reference.

	cmd := &cobra.Command{}
	args := []string{}

	// Capture the standard output
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	updater.Run(cmd, args)

	// Restore the real stdout
	w.Close()
	os.Stdout = old
	var buf strings.Builder
	io.Copy(&buf, r)

	expectedOutput := "Hyphen CLI updated successfully\n"
	assert.Contains(t, buf.String(), expectedOutput, "The output should indicate that the update was successful")
	assert.FileExists(t, mockFile.Name(), "The expected file should exist after update")
}

func TestUpdater_Run_InvalidOS(t *testing.T) {
	cliVersion.Version = "1.0.0" // Set current version for comparison

	mockHTTPClient := MockHTTPClient{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"version":"2.0.0"}]}`)),
		},
		Err: nil,
	}
	updater := Updater{
		Version:           "latest",
		BaseURL:           "https://api.hyphen.ai",
		URLTemplate:       "https://api.hyphen.ai/api/downloads/hyphen-cli/%s?os=%s",
		HTTPClient:        mockHTTPClient,
		FileHandler:       DefaultFileHandler{},
		GetExecPath:       defaultGetExecutablePath,
		DetectPlatform:    func() string { return "invalidOS" },
		DownloadAndUpdate: nil,
	}

	updater.DownloadAndUpdate = updater.downloadAndUpdate // Initialize the function reference.

	cmd := &cobra.Command{}
	args := []string{}

	errFunc := func() {
		updater.Run(cmd, args)
	}
	assert.NotPanics(t, errFunc, "The Run method should not panic for invalid OS")
}

func TestUpdater_DownloadAndUpdate_NetworkError(t *testing.T) {
	mockHTTPClient := MockHTTPClient{Response: nil, Err: errors.New("network error")}
	updater := Updater{
		Version:           "latest",
		BaseURL:           "https://api.hyphen.ai",
		URLTemplate:       "https://api.hyphen.ai/api/downloads/hyphen-cli/%s?os=%s",
		HTTPClient:        mockHTTPClient,
		FileHandler:       DefaultFileHandler{},
		GetExecPath:       defaultGetExecutablePath,
		DownloadAndUpdate: nil,
	}

	updater.DownloadAndUpdate = updater.downloadAndUpdate // Initialize the function reference.

	err := updater.downloadAndUpdate("https://api.hyphen.ai/api/downloads/hyphen-cli/latest?os=linux")
	assert.Error(t, err, "An error was expected due to network issue")
}

func TestUpdater_ScheduleWindowsUpdate_WriteError(t *testing.T) {
	mockFileHandler := MockFileHandler{
		WriteFileErr: errors.New("write file error"),
	}

	updater := Updater{
		Version:           "latest",
		BaseURL:           "https://api.hyphen.ai",
		URLTemplate:       "https://api.hyphen.ai/api/downloads/hyphen-cli/%s?os=%s",
		HTTPClient:        DefaultHTTPClient{},
		FileHandler:       mockFileHandler,
		GetExecPath:       defaultGetExecutablePath,
		DownloadAndUpdate: nil,
	}

	updater.DownloadAndUpdate = updater.downloadAndUpdate // Initialize the function reference.

	err := updater.scheduleWindowsUpdate("tempFileName")
	assert.Error(t, err, "An error was expected due to write file issue")
}

func TestDefaultHTTPClient_Get(t *testing.T) {
	client := DefaultHTTPClient{}
	url := "http://www.example.com"
	resp, err := client.Get(url)

	assert.NoError(t, err, "Expected no error for a valid URL")
	assert.NotNil(t, resp, "Expected a response for a valid URL")
	if resp != nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code to be 200")
	}
}

func TestDefaultFileHandler_CreateTemp(t *testing.T) {
	handler := DefaultFileHandler{}
	tempFile, err := handler.CreateTemp("", "hyphen_test")

	assert.NoError(t, err, "Expected no error creating a temporary file")
	assert.NotNil(t, tempFile, "Expected a valid temporary file")

	defer os.Remove(tempFile.Name())
}

func TestDefaultFileHandler_WriteFile(t *testing.T) {
	handler := DefaultFileHandler{}
	filename := "test_write_file.txt"
	content := []byte("test content")

	err := handler.WriteFile(filename, content, 0644)
	assert.NoError(t, err, "Expected no error writing to file")

	defer os.Remove(filename)
}

func TestDefaultFileHandler_Chmod(t *testing.T) {
	handler := DefaultFileHandler{}
	filename := "test_chmod_file.txt"
	content := []byte("test content")

	os.WriteFile(filename, content, 0644)

	err := handler.Chmod(filename, 0755)
	assert.NoError(t, err, "Expected no error changing file permissions")

	defer os.Remove(filename)
}

func TestDefaultFileHandler_Rename(t *testing.T) {
	handler := DefaultFileHandler{}
	oldpath := "old_file.txt"
	newpath := "new_file.txt"
	content := []byte("test content")

	os.WriteFile(oldpath, content, 0644)

	err := handler.Rename(oldpath, newpath)
	assert.NoError(t, err, "Expected no error renaming file")

	defer os.Remove(newpath)
}

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		goos     string
		goarch   string
		expected string
	}{
		{"linux", "", linux},
		{"darwin", "amd64", macos},
		{"darwin", "arm64", macosArm},
		{"windows", "", windows},
		{"freebsd", "", "freebsd"},
	}

	for _, tt := range tests {
		result := detectPlatform(tt.goos, tt.goarch)
		assert.Equal(t, tt.expected, result, "Expected platform: %s, but got: %s", tt.expected, result)
	}
}
