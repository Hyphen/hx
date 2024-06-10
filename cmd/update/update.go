package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
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

	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(filename, 0755); err != nil {
			return fmt.Errorf("error setting executable permission: %w", err)
		}
	}

	return nil
}

func init() {
	UpdateCmd.Flags().StringVar(&version, "version", "", "Specific version to update to (default is latest)")
}
