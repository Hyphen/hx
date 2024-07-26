package run

import (
	"fmt"

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

Warning:
  If the specified environment file does not exist, environment variables will be streamed.

Examples:
  # Run a command using the default environment
  hyrule env run default go run main.go

  # Run a command using a specific environment
  hyrule env run production some_script.sh

  # Run a command using a custom environment file
  hyrule env run staging --file .env.staging node server.js

  # Run a command while streaming environment variables
  hyrule env run development --stream python app.py

  # Run a command with additional arguments
  hyrule env run test npm run test -- --watch`,
	Args: cobra.MinimumNArgs(2), // Ensure at least two arguments are provided: ENVIRONMENT and COMMAND
	Run:  runCommand,
}

func runCommand(cmd *cobra.Command, args []string) {
	runCommander := InitRunCommander()
	env := args[0]
	command := args[1]
	commandArgs := args[2:]

	envFile := getEnvFile(env)

	if !fileExists(envFile) {
		StreamVars = true
	}

	envVars, err := runCommander.getEnvironmentVariables(env)
	if err != nil {
		fmt.Printf("Error exporting environment variables: %v\n", err)
		return
	}

	if err := runCommander.execute(command, commandArgs, envVars); err != nil {
		fmt.Printf("Error executing command '%s': %v\n", command, err)
	}
}

func init() {
	RunCmd.Flags().StringVarP(&envFile, "file", "f", "", "Specify a custom environment file (e.g., .env.prod or config.env)")
	RunCmd.Flags().BoolVarP(&StreamVars, "stream", "s", false, "Stream environment variables from the ENV service")
}
