package merge

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

var MergeCmd = &cobra.Command{
	Use:   "merge [ENVIRONMENT] [FILE]",
	Short: "Merge environment variables into a file",
	Long: `This command reads the specified environment, decrypts the variables, 
and merges them into the given file, giving preference to the pulled variables. The environment should be pushed first`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		env := args[0]
		file := args[1]

		envHandler := environment.Restore()

		// Decrypt environment variables
		decryptedVars, err := envHandler.DecryptEnvironmentVars(env)
		if err != nil {
			fmt.Printf("Error decrypting environment variables: %v\n", err)
			os.Exit(1)
		}

		// Read existing variables from the file
		existingVars, err := readEnvFile(file)
		if err != nil {
			fmt.Printf("Error reading from file %s: %v\n", file, err)
			os.Exit(1)
		}

		// Merge variables, with preference to the pulled variables
		mergedVars := mergeEnvVars(existingVars, decryptedVars)

		// Write merged variables back to the file
		err = writeEnvFile(file, mergedVars)
		if err != nil {
			fmt.Printf("Error writing to file %s: %v\n", file, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully merged environment variables to file %s\n", file)
	},
}

func init() {
}

// readEnvFile reads environment variables from a file into a map
func readEnvFile(fileName string) (map[string]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envVars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			envVars[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envVars, nil
}

// mergeEnvVars merges two sets of environment variables, with preference to the new variables
func mergeEnvVars(existingVars map[string]string, newVars []string) map[string]string {
	mergedVars := existingVars

	for _, line := range newVars {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Override existing variable with the new one
			mergedVars[key] = value
		}
	}

	return mergedVars
}

// writeEnvFile writes environment variables from a map to a file
func writeEnvFile(fileName string, envVars map[string]string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating or opening file: %w", err)
	}
	defer file.Close()

	for key, value := range envVars {
		_, err := file.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		if err != nil {
			return fmt.Errorf("error writing environment variables to file: %w", err)
		}
	}

	return nil
}
