package initialize

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func addAlias(alias, command string) error {
	shell := detectShell()
	configFile, err := getShellConfigFile(shell)
	if err != nil {
		return err
	}

	aliasCommand := fmt.Sprintf("alias %s='%s'", alias, command)
	if err := appendIfNotExists(configFile, aliasCommand); err != nil {
		return err
	}

	return sourceConfigFile(shell, configFile)
}

func detectShell() string {
	if runtime.GOOS == "windows" {
		// Check if we're running in PowerShell
		if os.Getenv("PSModulePath") != "" {
			return "powershell"
		}
		// Check for PowerShell Core
		if os.Getenv("PWSH_DISTRIBUTION_CHANNEL") != "" {
			return "powershell"
		}
		// If not PowerShell, assume it's CMD
		return "cmd"
	}

	// For non-Windows systems, use the existing logic
	shell := os.Getenv("SHELL")
	return filepath.Base(shell)
}

func getShellConfigFile(shell string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch shell {
	case "bash":
		return filepath.Join(homeDir, ".bashrc"), nil
	case "zsh":
		return filepath.Join(homeDir, ".zshrc"), nil
	case "ksh":
		return filepath.Join(homeDir, ".kshrc"), nil
	case "fish":
		return filepath.Join(homeDir, ".config", "fish", "config.fish"), nil
	case "powershell":
		profilePath := filepath.Join(homeDir, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		// Check if the directory exists, create it if it doesn't
		dir := filepath.Dir(profilePath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory for PowerShell profile: %v", err)
			}
		}
		// Create the file if it doesn't exist
		if _, err := os.Stat(profilePath); os.IsNotExist(err) {
			file, err := os.Create(profilePath)
			if err != nil {
				return "", fmt.Errorf("failed to create PowerShell profile file: %v", err)
			}
			file.Close()
		}
		return profilePath, nil
	case "cmd":
		return "", fmt.Errorf("cmd does not support aliases directly. Use a batch file instead.")
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

func appendIfNotExists(filename, text string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == text {
			return nil
		}
	}

	if _, err := file.WriteString(fmt.Sprintf("\n%s\n", text)); err != nil {
		return err
	}
	return nil
}

func sourceConfigFile(shell, configFile string) error {
	var cmd *exec.Cmd

	switch shell {
	case "bash", "zsh", "ksh":
		cmd = exec.Command(shell, "-c", fmt.Sprintf("source %s", configFile))
	case "fish":
		cmd = exec.Command("fish", "-c", fmt.Sprintf("source %s", configFile))
	case "powershell":
		cmd = exec.Command("powershell", "-Command", fmt.Sprintf(". '%s'", configFile))
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
