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
