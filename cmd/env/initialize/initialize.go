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

		defaultAppId := generateDefaultAppId(appName)
		appId, err := PromptForAppId(os.Stdin, defaultAppId)
		if err != nil {
			fmt.Println(err)
			return
		}

		_ = environment.Initialize(appName, appId)
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

func PromptForAppId(reader io.Reader, defaultAppId string) (string, error) {
	r := bufio.NewReader(reader)
	for {
		fmt.Printf("App ID [%s]: ", defaultAppId)
		appId, err := r.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("error reading input: %w", err)
		}
		appId = strings.TrimSpace(appId)
		if appId == "" {
			appId = defaultAppId
		}

		if err := environment.CheckAppId(appId); err != nil {
			fmt.Printf("Invalid App ID: %v\n", err)
			fmt.Println("Please try again.")
			continue
		}
		return appId, nil
	}
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

func generateDefaultAppId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}
