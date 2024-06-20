package decrypt

import (
	"fmt"
	"github.com/spf13/cobra"
)

var DecryptCmd = &cobra.Command{
	Use:     "decrypt [FILE]",
	Aliases: []string{"d"},
	Short:   "Decrypt a raw data dump",
	Long:    `Decrypts a raw data dump from CloudEnv, showing the decrypted environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Usage: cloudenv decrypt [FILE]")
			return
		}
		filename := args[0]
		fmt.Printf("Decrypted data from file %s.\n", filename)
	},
}
