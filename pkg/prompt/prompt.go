package prompt

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
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
