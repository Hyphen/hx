package run

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/environment"
)

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
