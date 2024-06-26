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
	envFile    string
	StreamVars bool
)

var RunCmd = &cobra.Command{
	Use:   "run [ENVIRONMENT] [COMMAND] [ARGS...]",
	Short: "Run a command using some environment variables",
	Long: `Executes the specified command with the environment variables sourced from the specified environment file.

Example usage:
  hyrule env run default go run main.go
  hyrule env run production some_script.sh`,
	Args: cobra.MinimumNArgs(2), // Ensure at least two arguments are provided: ENVIRONMENT and COMMAND
	Run:  runCommand,
}

func init() {
	RunCmd.Flags().StringVarP(&envFile, "file", "f", "", "specific environment file to use")
	RunCmd.Flags().BoolVarP(&StreamVars, "stream", "s", false, "stream environment variables")
}

func runCommand(cmd *cobra.Command, args []string) {
	runCommander := InitRunCommander()
	env := args[0]
	command := args[1]
	commandArgs := args[2:]

	envVars, err := runCommander.getEnvironmentVariables(env)
	if err != nil {
		fmt.Printf("Error exporting environment variables: %v\n", err)
		return
	}

	if err := runCommander.execute(command, commandArgs, envVars); err != nil {
		fmt.Printf("Error executing command '%s': %v\n", command, err)
	}
}

type RunCommander struct {
	envHanler environment.EnviromentHandler
}

func InitRunCommander() *RunCommander {
	return &RunCommander{
		envHanler: environment.Restore(),
	}
}

func (r *RunCommander) getEnvironmentVariables(env string) ([]string, error) {
	if StreamVars {
		return r.readEnvFileStremed(env)
	}
	return readEnvFile(env)
}

func (r *RunCommander) readEnvFileStremed(env string) ([]string, error) {
	return r.envHanler.DecryptEnvironmentVars(env)
}

func (r *RunCommander) execute(command string, args []string, envVars []string) error {
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

func getEnvFile(env string) string {
	if envFile == "" {
		return environment.GetEnvFileByEnvironment(env)
	}
	return envFile
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func readEnvFile(env string) ([]string, error) {
	envFile := getEnvFile(env)

	if !fileExists(envFile) {
		fmt.Printf("Environment file %s does not exist.\n", envFile)
		os.Exit(1)
	}

	f, err := os.Open(envFile)
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

func expandArgs(args []string) []string {
	expandedArgs := make([]string, len(args))
	for i, arg := range args {
		expandedArgs[i] = os.ExpandEnv(arg)
	}
	return expandedArgs
}
