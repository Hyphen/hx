package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Service interfaces for dependency injection
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

type FileHandler interface {
	CreateTemp(dir, pattern string) (*os.File, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Chmod(name string, mode os.FileMode) error
	Rename(oldpath, newpath string) error
}

// Default implementations
type DefaultHTTPClient struct{}
type DefaultFileHandler struct{}

func (d DefaultHTTPClient) Get(url string) (*http.Response, error) {
	return http.Get(url)
}

func (d DefaultFileHandler) CreateTemp(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

func (d DefaultFileHandler) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (d DefaultFileHandler) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (d DefaultFileHandler) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// CommandRunner is a function type for running commands, to facilitate mocking in tests
type CommandRunner func(name string, arg ...string) *exec.Cmd

// Config struct for dependencies and settings
type Updater struct {
	Version           string
	URLTemplate       string
	HTTPClient        HTTPClient
	FileHandler       FileHandler
	GetExecPath       func() string
	DetectPlatform    func() string
	DownloadAndUpdate func(url string) error
	CommandRunner     CommandRunner
}

const (
	linux    = "linux"
	macos    = "macos"
	macosArm = "macos-arm"
	windows  = "windows"
)

var validOs = []string{linux, macos, macosArm, windows}

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the Hyphen CLI",
	Long:  `This command updates the Hyphen CLI to the specified version or the latest version available for your operating system`,
	Run: func(cmd *cobra.Command, args []string) {
		updater := NewDefaultUpdater(version)
		updater.Run(cmd, args)
	},
}

func NewDefaultUpdater(version string) *Updater {
	// Check if the environment variable is set, else use the default URL
	defaultUrlTemplate := os.Getenv("HYPHEN_ENGINE_URL")
	if defaultUrlTemplate == "" {
		defaultUrlTemplate = "https://api.hyphen.ai/api/downloads/hyphen-cli/%s?os=%s"
	} else {
		defaultUrlTemplate += "/api/downloads/hyphen-cli/%s?os=%s"
	}

	updater := &Updater{
		Version:           version,
		URLTemplate:       defaultUrlTemplate,
		HTTPClient:        DefaultHTTPClient{},
		FileHandler:       DefaultFileHandler{},
		GetExecPath:       defaultGetExecutablePath,
		DetectPlatform:    defaultDetectPlatformWrapper,
		DownloadAndUpdate: nil,
		CommandRunner:     exec.Command, // Use exec.Command by default
	}
	updater.DownloadAndUpdate = updater.downloadAndUpdate // Initialize the function reference.
	return updater
}

func (u *Updater) Run(cmd *cobra.Command, args []string) {
	osType := u.DetectPlatform()
	if !isValidOs(osType) {
		fmt.Printf("Unsupported operating system: %s\n", osType)
		return
	}

	targetVersion := getTargetVersion(u.Version)
	updateUrl := fmt.Sprintf(u.URLTemplate, targetVersion, osType)
	err := u.DownloadAndUpdate(updateUrl)
	if err != nil {
		fmt.Printf("Failed to update Hyphen CLI: %v\n", err)
		return
	}
	fmt.Println("Hyphen CLI updated successfully")
}

func detectPlatform(goos, goarch string) string {
	switch goos {
	case "linux":
		return linux
	case "darwin":
		if goarch == "arm64" {
			return macosArm
		}
		return macos
	case "windows":
		return windows
	default:
		return goos
	}
}

func defaultDetectPlatformWrapper() string {
	return detectPlatform(runtime.GOOS, runtime.GOARCH)
}

func isValidOs(osType string) bool {
	for _, valid := range validOs {
		if osType == valid {
			return true
		}
	}
	return false
}

func getTargetVersion(version string) string {
	if strings.TrimSpace(version) == "" {
		return "latest"
	}
	return version
}

func (u *Updater) downloadAndUpdate(url string) error {
	resp, err := u.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching the update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download update, status code: %d", resp.StatusCode)
	}

	filename := "hyphen"
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	tempFile, err := u.FileHandler.CreateTemp("", filename)
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing to temp file: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := u.FileHandler.Chmod(tempFile.Name(), 0755); err != nil {
			return fmt.Errorf("error setting executable permission: %w", err)
		}
		return u.moveToExecutablePath(tempFile.Name())
	}

	return u.scheduleWindowsUpdate(tempFile.Name())
}

func (u *Updater) scheduleWindowsUpdate(tempFileName string) error {
	executablePath := u.GetExecPath()
	batchScript := `
@echo off
echo Updating Hyphen CLI...
ping 127.0.0.1 -n 5 > nul
move /Y "%s" "%s"
if %%errorlevel%% neq 0 (
    echo Failed to move updated file.
    exit /b %%errorlevel%%
) else (
    echo Successfully moved updated file.
)
`
	scriptContent := fmt.Sprintf(batchScript, tempFileName, executablePath)
	scriptPath := filepath.Join(os.TempDir(), "update_hyphen.bat")

	if err := u.FileHandler.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		return fmt.Errorf("error writing batch script: %w", err)
	}

	cmd := u.CommandRunner("cmd", "/C", "start", "/MIN", scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting batch script: %w", err)
	}

	fmt.Println("Update scheduled. The CLI will be updated after it exits.")
	return nil
}

func (u *Updater) moveToExecutablePath(src string) error {
	executablePath := u.GetExecPath()
	return u.FileHandler.Rename(src, executablePath)
}

func defaultGetExecutablePath() string {
	path, err := os.Executable()
	if err != nil {
		fmt.Printf("Could not determine executable path: %v\n", err)
		os.Exit(1)
	}
	return path
}

func init() {
	UpdateCmd.Flags().StringVar(&version, "version", "", "Specific version to update to (default is latest)")
}

var version string
