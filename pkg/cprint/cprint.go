package cprint

import (
	"fmt"

	"github.com/Hyphen/cli/pkg/flags"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	green  = color.New(color.FgGreen, color.Bold).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	white  = color.New(color.FgWhite, color.Bold).SprintFunc()
	red    = color.New(color.FgRed, color.Bold).SprintFunc()
)

// Error prints an error message in red
func Error(cmd *cobra.Command, err error) {
	if flags.VerboseFlag {
		errorDetails := color.New(color.FgWhite).SprintFunc()
		cmd.PrintErrf("%s %s\n", red("ERROR:"), errorDetails(err.Error()))
	} else {
		fmt.Println("ERROR:", err.Error())
	}
}

// Info prints an informational message
func Info(message string) {
	if flags.VerboseFlag {
		fmt.Printf("%s %s\n", cyan("ℹ"), white(message))
	} else {
		fmt.Println(message)
	}
}

func YellowPrint(message string) {
	if flags.VerboseFlag {
		fmt.Printf("%s\n", yellow(message))
	} else {
		fmt.Println(message)
	}
}

func GreenPrint(message string) {
	if flags.VerboseFlag {
		fmt.Printf("%s\n", green(message))
	} else {
		fmt.Println(message)
	}
}

func Print(message string) {
	if flags.VerboseFlag {
		fmt.Printf("%s\n", white(message))
	} else {
		fmt.Println(message)
	}
}

// PrintNorm prints a normal message without any special formatting or symbols
func PrintNorm(message string) {
	fmt.Println(message)
}

// Success prints a success message
func Success(message string) {
	if flags.VerboseFlag {
		fmt.Printf("%s %s\n", green("✅"), white(message))
	} else {
		fmt.Println(message)
	}
}

// Warning prints a warning message
func Warning(message string) {
	if flags.VerboseFlag {
		fmt.Printf("%s %s\n", yellow("⚠"), white(message))
	} else {
		fmt.Println("WARNING:", message)
	}
}

// OrganizationInfo prints organization information
func OrganizationInfo(orgID string) {
	if flags.VerboseFlag {
		fmt.Printf("\n%s\n", white("Organization Information:"))
		fmt.Printf("  %s %s\n", white("ID:"), cyan(orgID))
	} else {
		fmt.Println("Organization ID:", orgID)
	}
}

// Prompt prints a prompt message
func Prompt(message string) {
	if flags.VerboseFlag {
		fmt.Print(yellow(message))
	} else {
		fmt.Print(message)
	}
}

// PrintHeader prints a header
func PrintHeader(message string) {
	if flags.VerboseFlag {
		fmt.Printf("\n%s\n", yellow(message))
	} else {
		fmt.Println(message)
	}
}

// PrintDetail prints a detail line
func PrintDetail(label, value string) {
	if flags.VerboseFlag {
		fmt.Printf("   %s %s\n", white(label+":"), cyan(value))
	} else {
		fmt.Printf("%s: %s\n", label, value)
	}
}
