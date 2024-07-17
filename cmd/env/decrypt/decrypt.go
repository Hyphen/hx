package decrypt

import (
	"fmt"
	"io"
	"os"

	"github.com/Hyphen/cli/internal/environment"
	"github.com/spf13/cobra"
)

var decryptString string
var decryptFile string

var DecryptCmd = &cobra.Command{
	Use:     "decrypt",
	Aliases: []string{"d"},
	Short:   "Decrypt a raw data dump",
	Long:    `Decrypts a raw data dump from Hyphen, showing the decrypted environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		if decryptString == "" && decryptFile == "" {
			fmt.Println("Usage: hyphen env decrypt -s [STRING] or hyphen env decrypt -f [FILE]")
			return
		}
		if decryptString != "" && decryptFile != "" {
			fmt.Println("Please provide only one of -s or -f, not both")
			return
		}

		envHandler := environment.Restore()

		var decrypted string
		var err error

		if decryptString != "" {
			decrypted, err = envHandler.SecretKey().Decrypt(decryptString)
			if err != nil {
				fmt.Printf("Error decrypting string: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Decrypted data:\n%s\n", decrypted)
		} else if decryptFile != "" {
			fileContent, err := readFileContent(decryptFile)
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				os.Exit(1)
			}

			decrypted, err = envHandler.SecretKey().Decrypt(fileContent)
			if err != nil {
				fmt.Printf("Error decrypting file content: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Decrypted data from file %s:\n%s\n", decryptFile, decrypted)
		}
	},
}

func init() {
	DecryptCmd.Flags().StringVarP(&decryptString, "string", "s", "", "String to decrypt")
	DecryptCmd.Flags().StringVarP(&decryptFile, "file", "f", "", "File to decrypt")
}

func readFileContent(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
