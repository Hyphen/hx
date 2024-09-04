package initialize

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var appIDFlag string

var InitCmd = &cobra.Command{
	Use:   "init <app name>",
	Short: "Initialize a new app",
	Long: `The 'hyphen init' command initializes a new app within your organization.

You need to provide an app name as a positional argument. Optionally, you can specify a custom app ID using the '--id' flag. If no app ID is provided, a default ID will be generated based on the app name.

If a manifest file already exists, you will be prompted to confirm if you want to overwrite it, unless the '--yes' flag is provided, in which case the manifest file will be overwritten automatically.

Example usages:
  hyphen init MyApp
  hyphen init MyApp --id custom-app-id --yes

Flags:
  --id, -i   Specify a custom app ID (optional)
  --yes, -y  Automatically confirm prompts and overwrite files if necessary`,
	Args: cobra.ExactArgs(1),
	Run:  runInit,
}

// Color definitions
var (
	green  = color.New(color.FgGreen, color.Bold).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	white  = color.New(color.FgWhite, color.Bold).SprintFunc()
)

func init() {
	InitCmd.Flags().StringVarP(&appIDFlag, "id", "i", "", "App ID (optional)")
}

func runInit(cmd *cobra.Command, args []string) {
	appService := app.NewService()
	orgID, err := utils.GetOrganizationID()
	if err != nil {
		errors.PrintError(cmd, err)
		return
	}

	appName := args[0]
	if appName == "" {
		errors.PrintError(cmd, fmt.Errorf("app name is required"))
		return
	}

	appAlternateId := getAppID(cmd, appName)
	if appAlternateId == "" {
		return
	}

	if manifest.Exists() && !shouldOverwrite(cmd) {
		printInfo(cmd, "Operation cancelled.")
		return
	}

	newApp, err := appService.CreateApp(orgID, appAlternateId, appName)
	if err != nil {
		errors.PrintError(cmd, err)
		os.Exit(1)
	}

	_, err = manifest.Initialize(orgID, newApp.Name, newApp.ID, newApp.AlternateId)
	if err != nil {
		errors.PrintError(cmd, err)
		os.Exit(1)
	}

	printSuccess(cmd, "App initialized")

	if err := ensureGitignore(manifest.ManifestConfigFile); err != nil {
		errors.PrintError(cmd, fmt.Errorf("error checking/updating .gitignore: %w", err))
		os.Exit(1)
	}
}

func getAppID(cmd *cobra.Command, appName string) string {
	defaultAppAlternateId := generateDefaultAppId(appName)
	appAlternateId := appIDFlag
	if appAlternateId == "" {
		appAlternateId = defaultAppAlternateId
	}

	err := app.CheckAppId(appAlternateId)
	if err != nil {
		suggestedID := strings.TrimSpace(strings.Split(err.Error(), ":")[1])
		yesFlag, _ := cmd.Flags().GetBool("yes")
		if yesFlag {
			appAlternateId = suggestedID
			printInfo(cmd, "Using suggested app ID: %s", suggestedID)
		} else {
			if !promptForSuggestedID(os.Stdin, suggestedID) {
				printInfo(cmd, "Operation cancelled.")
				return ""
			}
			appAlternateId = suggestedID
		}
	}
	return appAlternateId
}

func shouldOverwrite(cmd *cobra.Command) bool {
	yesFlag, _ := cmd.Flags().GetBool("yes")
	if yesFlag {
		return true
	}
	return promptForOverwrite(os.Stdin)
}

func generateDefaultAppId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}

func promptForOverwrite(reader io.Reader) bool {
	r := bufio.NewReader(reader)
	for {
		fmt.Print(yellow("Manifest file exists. Do you want to overwrite it? (y/N): "))
		response, err := r.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
			os.Exit(1)
		}
		response = strings.TrimSpace(response)
		switch strings.ToLower(response) {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Println("Invalid response. Please enter 'y' or 'n'.")
		}
	}
}

func promptForSuggestedID(reader io.Reader, suggestedID string) bool {
	r := bufio.NewReader(reader)
	for {
		fmt.Printf(yellow("Invalid app ID. Do you want to use the suggested ID [%s]? (Y/n): "), cyan(suggestedID))
		response, err := r.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
			os.Exit(1)
		}
		response = strings.TrimSpace(response)
		switch strings.ToLower(response) {
		case "y", "yes", "":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Invalid response. Please enter 'y' or 'n'.")
		}
	}
}

func printInfo(cmd *cobra.Command, format string, a ...interface{}) {
	cmd.Printf(white(format)+"\n", a...)
}

func printSuccess(cmd *cobra.Command, format string, a ...interface{}) {
	cmd.Printf(green("✅ ")+white(format)+"\n", a...)
}
