package initialize

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"i"},
	Short:   "Initialize the environment",
	Long:    `This command initializes the environment with necessary configurations.`,
	Run: func(cmd *cobra.Command, args []string) {

		if environment.ConfigExists() {
			if !PromptForOverwrite(os.Stdin) {
				fmt.Println("Operation cancelled.")
				return
			}
		}

		appName, err := PromptForAppName(os.Stdin)
		if err != nil {
			fmt.Println(err)
			return
		}
		//TODO:
		//check if name exist
		//xxx

		environment.Initialize(appName)
		fmt.Println("Environment initialized")

		if err := environment.EnsureGitignore(); err != nil {
			fmt.Printf("Error checking/updating .gitignore: %v\n", err)
			os.Exit(1)
		}
	},
}

func PromptForAppName(reader io.Reader) (string, error) {
	r := bufio.NewReader(reader)
	fmt.Print("Name of App: ")
	appName, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading input: %w", err)
	}
	return strings.TrimSpace(appName), nil
}

func PromptForOverwrite(reader io.Reader) bool {
	r := bufio.NewReader(reader)
	for {
		fmt.Print("Are you sure you want to overwrite the EnvConfigFile? (y/N): ")
		response, err := r.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			os.Exit(1)
		}
		response = strings.TrimSpace(response)
		switch strings.ToLower(response) {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
		}
	}
}
