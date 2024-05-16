package auth

import (
	"fmt"

	"github.com/Hyphen/cli/config"
	"github.com/spf13/cobra"
)

// AuthCmd represents the auth command
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the server",
	Long: `Authenticate allows users to start a session with the server
by providing credentials.`,
	Run: func(cmd *cobra.Command, args []string) {
		login()
	},
}

func login() {
	var username, password string
	fmt.Print("Enter Username: ")
	fmt.Scanln(&username)
	fmt.Print("Enter Password: ")
	fmt.Scanln(&password)

	fmt.Println("Logging in with Username:", username, "and Password:", password)
	config.SaveCredentials(username, password)
	fmt.Println("Login successful")
}
