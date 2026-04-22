package cprint

import (
	"encoding/json"
	"fmt"
	"sync"

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

// FormatJSON selects structured JSON output. Callers set it via SetFormat on
// a CPrinter. Consumers wrapping the CLI (e.g. the deploy-action GitHub
// wrapper) rely on this contract to parse results.
const FormatJSON = "json"

// Message levels recorded in the JSON payload.
const (
	LevelInfo    = "info"
	LevelSuccess = "success"
	LevelWarning = "warning"
	LevelError   = "error"
	LevelVerbose = "verbose"
)

// CPrinter holds the configuration for printing. Safe for concurrent use —
// callbacks from socket.io event streams call the printer from goroutines
// other than the main one, so the messages buffer needs a mutex.
type CPrinter struct {
	verbose  bool
	format   string
	mu       sync.Mutex
	messages []Message
}

// Message is a user-facing line recorded by the printer. In human mode it
// streams to stdout as it happens. In json mode it is buffered and emitted
// as part of the final JSON object produced by Emit.
type Message struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// NewCPrinter creates a new CPrinter instance
func NewCPrinter(verbose bool) *CPrinter {
	return &CPrinter{
		verbose: verbose,
	}
}

// SetFormat selects how this printer renders output. The zero value (empty
// string) is the default human-readable mode. FormatJSON suppresses
// streaming progress and buffers messages into the JSON payload emitted by
// Emit.
func (p *CPrinter) SetFormat(format string) {
	p.format = format
}

// IsJSON reports whether the printer is in structured JSON mode.
func (p *CPrinter) IsJSON() bool {
	return p.format == FormatJSON
}

// Messages returns a copy of the messages recorded so far. Exposed mostly
// for tests; returning a copy keeps callers from mutating printer state.
func (p *CPrinter) Messages() []Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Message, len(p.messages))
	copy(out, p.messages)
	return out
}

// Emit renders the final output of a command.
//
//   - In human mode it is a no-op — individual messages have already been
//     streamed to stdout as they happened.
//   - In json mode it merges the caller's result fields with a "messages"
//     array of everything recorded during the run and writes a single JSON
//     object to stdout on its own line.
//
// The caller owns the result keys. A "messages" key in result is
// overwritten by the printer's recorded messages — the CLI owns that slot.
func (p *CPrinter) Emit(result map[string]any) error {
	if !p.IsJSON() {
		return nil
	}

	payload := make(map[string]any, len(result)+1)
	for k, v := range result {
		payload[k] = v
	}

	p.mu.Lock()
	if p.messages == nil {
		payload["messages"] = []Message{}
	} else {
		msgs := make([]Message, len(p.messages))
		copy(msgs, p.messages)
		payload["messages"] = msgs
	}
	p.mu.Unlock()

	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	fmt.Println(string(encoded))
	return nil
}

func (p *CPrinter) record(level, text string) {
	p.mu.Lock()
	p.messages = append(p.messages, Message{Level: level, Text: text})
	p.mu.Unlock()
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

// CPrinter methods — each records the message and, in human mode only,
// streams it to stdout.
func (p *CPrinter) Error(cmd *cobra.Command, err error) {
	p.record(LevelError, err.Error())
	if p.IsJSON() {
		return
	}
	Error(cmd, err, true)
}

func (p *CPrinter) Info(message string) {
	p.record(LevelInfo, message)
	if p.IsJSON() {
		return
	}
	Info(message)
}

func (p *CPrinter) YellowPrint(message string) {
	p.record(LevelInfo, message)
	if p.IsJSON() {
		return
	}
	YellowPrint(message)
}

func (p *CPrinter) GreenPrint(message string) {
	p.record(LevelInfo, message)
	if p.IsJSON() {
		return
	}
	GreenPrint(message)
}

func (p *CPrinter) Print(message string) {
	p.record(LevelInfo, message)
	if p.IsJSON() {
		return
	}
	Print(message)
}

func (p *CPrinter) PrintNorm(message string) {
	p.record(LevelInfo, message)
	if p.IsJSON() {
		return
	}
	PrintNorm(message)
}

func (p *CPrinter) Success(message string) {
	p.record(LevelSuccess, message)
	if p.IsJSON() {
		return
	}
	Success(message)
}

func (p *CPrinter) Warning(message string) {
	p.record(LevelWarning, message)
	if p.IsJSON() {
		return
	}
	Warning(message)
}

func (p *CPrinter) OrganizationInfo(orgID string) {
	p.record(LevelInfo, "Organization ID: "+orgID)
	if p.IsJSON() {
		return
	}
	OrganizationInfo(orgID, p.verbose)
}

func (p *CPrinter) Prompt(message string) {
	// Prompts are interactive and don't make sense in json mode; record
	// them as info so consumers can see they happened.
	p.record(LevelInfo, message)
	if p.IsJSON() {
		return
	}
	Prompt(message, p.verbose)
}

func (p *CPrinter) PrintHeader(message string) {
	p.record(LevelInfo, message)
	if p.IsJSON() {
		return
	}
	PrintHeader(message)
}

func (p *CPrinter) PrintDetail(label, value string) {
	p.record(LevelInfo, label+": "+value)
	if p.IsJSON() {
		return
	}
	PrintDetail(label, value)
}

// PrintVerbose records and prints only when verbose mode is enabled.
func (p *CPrinter) PrintVerbose(message string) {
	if !p.verbose {
		return
	}
	p.record(LevelVerbose, message)
	if p.IsJSON() {
		return
	}
	Print(message)
}
