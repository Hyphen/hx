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

Example usage:
  hyrule env run default go run main.go
  hyrule env run production some_script.sh`,
	Args: cobra.MinimumNArgs(2), // Ensure at least two arguments are provided: ENVIRONMENT and COMMAND
	Run:  runCommand,
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

func init() {
	RunCmd.Flags().StringVarP(&envFile, "file", "f", "", "specific environment file to use")
	RunCmd.Flags().BoolVarP(&StreamVars, "stream", "s", false, "stream environment variables")
}
