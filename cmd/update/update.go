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

var version string
var defaultUrlTemplate = "http://localhost:4000/api/downloads/hyphen-cli/%s?os=%s"

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
		osType := detectPlatform()
		fmt.Println("Detected OS Type:")
		fmt.Println(osType)
		if !isValidOs(osType) {
			fmt.Printf("Unsupported operating system: %s\n", osType)
			return
		}

		// Get the target version (use "latest" if not specified)
		targetVersion := getTargetVersion()

		// Generate the update URL
		updateUrl := fmt.Sprintf(defaultUrlTemplate, targetVersion, osType)
		err := downloadAndUpdate(updateUrl)
		if err != nil {
			fmt.Printf("Failed to update Hyphen CLI: %v\n", err)
			return
		}

		fmt.Println("Hyphen CLI updated successfully")
	},
}

func detectPlatform() string {
	switch runtime.GOOS {
	case "linux":
		return linux
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return macosArm
		}
		return macos
	case "windows":
		return windows
	default:
		return runtime.GOOS
	}
}

func isValidOs(osType string) bool {
	for _, valid := range validOs {
		if osType == valid {
			return true
		}
	}
	return false
}

func getTargetVersion() string {
	// Use "latest" if no version is specified
	if strings.TrimSpace(version) == "" {
		return "latest"
	}
	return version
}

func downloadAndUpdate(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching the update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download update, status code: %d", resp.StatusCode)
	}

	filename := "hyphen-cli"
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	tempFile, err := os.CreateTemp("", filename)
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing to temp file: %w", err)
	}
	tempFile.Close()

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempFile.Name(), 0755); err != nil {
			return fmt.Errorf("error setting executable permission: %w", err)
		}
		return replaceFile(tempFile.Name(), filename)
	}

	// For Windows, use a batch script to replace the file after the process exits
	return scheduleWindowsUpdate(tempFile.Name())
}

func scheduleWindowsUpdate(tempFileName string) error {
	executablePath := getExecutablePath()
	batchScript := `
@echo off
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

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		return fmt.Errorf("error writing batch script: %w", err)
	}

	cmd := exec.Command("cmd", "/C", "start", "/MIN", scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting batch script: %w", err)
	}

	fmt.Println("Update scheduled. The CLI will be updated after it exits.")
	return nil
}

func getExecutablePath() string {
	path, err := os.Executable()
	if err != nil {
		fmt.Printf("Could not determine executable path: %v\n", err)
		os.Exit(1)
	}
	return path
}

func replaceFile(src, dst string) error {
	return os.Rename(src, dst)
}

func init() {
	UpdateCmd.Flags().StringVar(&version, "version", "", "Specific version to update to (default is latest)")
}
