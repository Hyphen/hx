package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type Response struct {
	Confirmed bool
	IsFlag    bool
}

func PromptYesNo(cmd *cobra.Command, prompt string, defaultValue bool) Response {
	yesFlag, _ := cmd.Flags().GetBool("yes")
	noFlag, _ := cmd.Flags().GetBool("no")

	var response string
	defaultStr := "Y/n"
	if !defaultValue {
		defaultStr = "y/N"
	}
	fmt.Printf("%s [%s]: ", prompt, defaultStr)
	if yesFlag {
		fmt.Println("y")
		return Response{Confirmed: true, IsFlag: true}
	} else if noFlag {
		fmt.Println("n")
		return Response{Confirmed: false, IsFlag: true}
	}

	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	switch strings.ToLower(strings.TrimSpace(response)) {
	case "y", "yes":
		return Response{Confirmed: true, IsFlag: false}
	case "n", "no":
		return Response{Confirmed: false, IsFlag: false}
	case "":
		return Response{Confirmed: defaultValue, IsFlag: false}
	default:
		cprint.Warning("Invalid response. Please enter 'y' or 'n'.")
		return PromptYesNo(cmd, prompt, defaultValue)
	}
}

func PromptString(cmd *cobra.Command, prompt string) (string, error) {
	fmt.Printf("%s ", prompt)

	noFlag, _ := cmd.Flags().GetBool("no")
	if noFlag {
		return "", fmt.Errorf("operation cancelled due to --no flag")
	}

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Trim the newline character at the end
	response = strings.TrimSpace(response)

	if response == "" {
		return "", fmt.Errorf("no response provided")
	}

	return response, nil
}

func PromptPassword(cmd *cobra.Command, prompt string) (string, error) {
	fmt.Print(prompt)
	noFlag, _ := cmd.Flags().GetBool("no")
	if noFlag {
		return "", fmt.Errorf("operation cancelled due to --no flag")
	}

	byteKey, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}
	fmt.Println()

	return string(byteKey), nil
}
