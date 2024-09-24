package cprint

import (
	"fmt"

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
	errorDetails := color.New(color.FgWhite).SprintFunc()
	cmd.PrintErrf("%s %s\n", red("ERROR:"), errorDetails(err.Error()))
}

// Info prints an informational message with a cyan info symbol
func Info(message string) {
	fmt.Printf("%s %s\n", cyan("ℹ"), white(message))
}

func YellowPrint(message string) {
	fmt.Printf("%s\n", yellow(message))
}

func GreenPrint(message string) {
	fmt.Printf("%s\n", green(message))
}

func Print(message string) {
	fmt.Printf("%s\n", white(message))
}

// PrintNorm prints a normal message without any special formatting or symbols
func PrintNorm(message string) {
	fmt.Println(message)
}

// Success prints a success message with a green checkmark
func Success(message string) {
	fmt.Printf("%s %s\n", green("✅"), white(message))
}

// Warning prints a warning message in yellow
func Warning(message string) {
	fmt.Printf("%s %s\n", yellow("⚠"), white(message))
}

// OrganizationInfo prints organization information
func OrganizationInfo(orgID string) {
	fmt.Printf("\n%s\n", white("Organization Information:"))
	fmt.Printf("  %s %s\n", white("ID:"), cyan(orgID))
}

// Prompt prints a prompt message in yellow
func Prompt(message string) {
	fmt.Print(yellow(message))
}

// PrintHeader prints a header in yellow
func PrintHeader(message string) {
	fmt.Printf("\n%s\n", yellow(message))
}

// PrintDetail prints a detail line with a white label and cyan value
func PrintDetail(label, value string) {
	fmt.Printf("   %s %s\n", white(label+":"), cyan(value))
}
