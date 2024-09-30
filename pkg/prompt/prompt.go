package prompt

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
)

func PromptYesNo(cmd *cobra.Command, prompt string, defaultValue bool) bool {
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
		return true
	} else if noFlag {
		fmt.Println("n")
		return false
	}

	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	switch strings.ToLower(strings.TrimSpace(response)) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	case "":
		return defaultValue
	default:
		cprint.Warning("Invalid response. Please enter 'y' or 'n'.")
		return PromptYesNo(cmd, prompt, defaultValue)
	}
}
