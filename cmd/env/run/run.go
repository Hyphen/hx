package run

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

var (
	envFile      string
	envWorkspace string
	StreamVars   bool
)

var RunCmd = &cobra.Command{
	Use:   "run [COMMAND]",
	Short: "Run a command using some environment variables",
	Long:  `Executes the specified command with the environment variables sourced from the specified environment file.`,
	Run:   runCommand,
}

func init() {
	RunCmd.Flags().StringVarP(&envFile, "file", "f", "", "specific environment file to use")
	RunCmd.Flags().StringVarP(&envWorkspace, "env", "e", "", "environment workspace (e.g., development, production)")
	RunCmd.Flags().BoolVarP(&StreamVars, "stream", "s", false, "stream environment variables")
}

func runCommand(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: cloudenv run [COMMAND]")
		return
	}

	command := args[0]
	env := getEnvFile()

	if !fileExists(env) {
		fmt.Printf("Environment file %s does not exist.\n", env)
		return
	}

	envVars, err := getEnvironmentVariables(env)
	if err != nil {
		fmt.Printf("Error exporting environment variables: %v\n", err)
		return
	}

	fmt.Printf("Running command '%s' with environment variables from %s.\n", command, env)
	if err := executeCommand(command, args[1:], envVars); err != nil {
		fmt.Printf("Error executing command '%s': %v\n", command, err)
	}
}

func getEnvFile() string {
	if envFile == "" {
		return environment.GetEnvFileByEnvironment(envWorkspace)
	}
	return envFile
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func getEnvironmentVariables(env string) ([]string, error) {
	if StreamVars {
		// TODO: Handle streaming environment variables
		return []string{}, nil // Placeholder for the actual streaming logic
	}
	return readEnvFile(env)
}

func readEnvFileStremed(env string) ([]string, error) {
	return nil, nil
}

func readEnvFile(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	var envVars []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if shouldIgnoreLine(line) {
			continue
		}

		key, value, err := parseEnvLine(line)
		if err != nil {
			return nil, err
		}

		os.Setenv(key, value)
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return envVars, nil
}

func shouldIgnoreLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

func parseEnvLine(line string) (string, string, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid line: %s", line)
	}
	return parts[0], parts[1], nil
}

func executeCommand(command string, args []string, envVars []string) error {
	cmd := exec.Command(os.ExpandEnv(command), expandArgs(args)...)
	cmd.Env = append(os.Environ(), envVars...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	return nil
}

func expandArgs(args []string) []string {
	expandedArgs := make([]string, len(args))
	for i, arg := range args {
		expandedArgs[i] = os.ExpandEnv(arg)
	}
	return expandedArgs
}
