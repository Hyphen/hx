package initialize

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// ConfigureCmd represents the configure command
var InitCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure environment variables",
	Long:  `Configure sets up environment variables and aliases for the CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := addAlias("envx", "hyphen env"); err != nil {
			fmt.Println("Error:", err)
		}
	},
}

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

	fmt.Printf("Alias '%s' added to %s\n", aliasCommand, configFile)
	return sourceConfigFile(shell, configFile)
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" && runtime.GOOS == "windows" {
		shell = os.Getenv("ComSpec")
		if strings.Contains(shell, "powershell") {
			return "powershell"
		}
		return "cmd"
	}
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
		return filepath.Join(homeDir, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"), nil
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
