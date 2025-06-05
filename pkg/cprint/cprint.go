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

// CPrinter holds the configuration for printing
type CPrinter struct {
	verbose bool
}

// NewCPrinter creates a new CPrinter instance
func NewCPrinter(verbose bool) *CPrinter {
	return &CPrinter{
		verbose: verbose,
	}
}

// Standalone functions
func Error(cmd *cobra.Command, err error, verbose bool) {
	if verbose {
		errorDetails := color.New(color.FgWhite).SprintFunc()

		cmd.PrintErrf("%s %s\n", red("❗error - "), errorDetails(err.Error()))
	} else {
		fmt.Println("ERROR:", err.Error())
	}
}

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

func PrintNorm(message string) {
	fmt.Println(message)
}

func Success(message string) {
	fmt.Printf("%s %s\n", green("✅"), white(message))
}

func Warning(message string) {
	fmt.Printf("%s %s\n", yellow("⚠"), white(message))
}

func OrganizationInfo(orgID string, verbose bool) {
	if verbose {
		fmt.Printf("\n%s\n", white("Organization Information:"))
		fmt.Printf("  %s %s\n", white("ID:"), cyan(orgID))
	} else {
		fmt.Println("Organization ID:", orgID)
	}
}

func Prompt(message string, verbose bool) {
	fmt.Print(yellow(message))
}

func PrintHeader(message string) {
	fmt.Printf("\n%s\n", yellow(message))
}

func PrintDetail(label, value string) {
	fmt.Printf("   %s %s\n", white(label+":"), cyan(value))
}

// CPrinter methods
func (p *CPrinter) Error(cmd *cobra.Command, err error) {
	Error(cmd, err, true)
}

func (p *CPrinter) Info(message string) {
	Info(message)
}

func (p *CPrinter) YellowPrint(message string) {
	YellowPrint(message)
}

func (p *CPrinter) GreenPrint(message string) {
	GreenPrint(message)
}

func (p *CPrinter) Print(message string) {
	Print(message)
}

func (p *CPrinter) PrintNorm(message string) {
	PrintNorm(message)
}

func (p *CPrinter) Success(message string) {
	Success(message) //issues/168 introduce always emoji
}

func (p *CPrinter) Warning(message string) {
	Warning(message)
}

func (p *CPrinter) OrganizationInfo(orgID string) {
	OrganizationInfo(orgID, p.verbose)
}

func (p *CPrinter) Prompt(message string) {
	Prompt(message, p.verbose)
}

func (p *CPrinter) PrintHeader(message string) {
	PrintHeader(message)
}

func (p *CPrinter) PrintDetail(label, value string) {
	PrintDetail(label, value)
}

func (p *CPrinter) PrintVerbose(message string) {
	if p.verbose {
		Print(message)
	}
}
