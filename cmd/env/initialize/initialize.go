package initialize

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/secretkey"
	"github.com/spf13/cobra"

	"github.com/BurntSushi/toml"
)

type Config struct {
	AppName   string `toml:"app_name"`
	SecretKey string `toml:"secret_key"`
}

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

		// Create and write to the .hyphen-env-key file
		config := Config{
			AppName:   appName,
			SecretKey: secretkey.New().Base64(),
		}

		file, err := os.Create(".hyphen-env-key")
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close()

		enc := toml.NewEncoder(file)
		if err := enc.Encode(config); err != nil {
			fmt.Println("Error encoding TOML:", err)
			return
		}

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
