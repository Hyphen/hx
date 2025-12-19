package prompt

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var printer *cprint.CPrinter

func init() {
	printer = cprint.NewCPrinter(flags.VerboseFlag)
}

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
		printer.Warning("Invalid response. Please enter 'y' or 'n'.")
		return PromptYesNo(cmd, prompt, defaultValue)
	}
}

func PromptString(cmd *cobra.Command, prompt string) (string, error) {
	fmt.Printf("%s ", prompt)

	noFlag, _ := cmd.Flags().GetBool("no")
	if noFlag {
		return "", fmt.Errorf("Operation cancelled due to --no flag")
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
		return "", fmt.Errorf("Operation cancelled due to --no flag")
	}

	byteKey, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}
	fmt.Println()

	return string(byteKey), nil
}

func PromptDir(cmd *cobra.Command, prompt string, mustExist bool) (string, error) {
	fmt.Printf("%s ", prompt)

	noFlag, _ := cmd.Flags().GetBool("no")
	if noFlag {
		return "", fmt.Errorf("Operation cancelled due to --no flag")
	}

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	response = strings.TrimSpace(response)

	if response == "" {
		return "", fmt.Errorf("no directory path provided")
	}

	if strings.HasPrefix(response, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error expanding home directory: %w", err)
		}
		response = strings.Replace(response, "~", homeDir, 1)
	}

	if mustExist {
		fileInfo, err := os.Stat(response)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("directory does not exist: %s", response)
			}
			return "", fmt.Errorf("error checking directory: %w", err)
		}

		if !fileInfo.IsDir() {
			return "", fmt.Errorf("path is not a directory: %s", response)
		}
	}

	return response, nil
}

func PromptForMonorepoApps(cmd *cobra.Command) ([]string, error) {
	// First prompt for the monorepo directory
	monorepoDir, err := PromptDir(cmd, "Enter the monorepo directory path:", true)
	if err != nil {
		return nil, err
	}

	// Read directory contents
	entries, err := os.ReadDir(monorepoDir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	// Filter only directories
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") { // Skip hidden directories
			dirs = append(dirs, entry.Name())
		}
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("no directories found in %s", monorepoDir)
	}

	fmt.Println("\nLet's go through each directory:")
	var selectedDirs []string
	for _, dir := range dirs {
		dirResponse := PromptYesNo(cmd, fmt.Sprintf("Include %s?", dir), false)
		if dirResponse.Confirmed {
			selectedDirs = append(selectedDirs, dir)
		}
	}

	// Convert to full paths
	var fullPaths []string
	for _, dir := range selectedDirs {
		fullPath := filepath.Join(monorepoDir, dir)
		fullPaths = append(fullPaths, fullPath)
	}

	if len(fullPaths) == 0 {
		return nil, fmt.Errorf("no directories were selected")
	}

	return fullPaths, nil
}

func PromptSelection(choices []Choice, prompt string) (Choice, error) {
	mod := model{
		choices: choices,
		cursor:  0,
		prompt:  prompt,
	}

	p := tea.NewProgram(mod)
	// Run returns the model as a tea.Model.
	m, err := p.Run()
	if err != nil {
		return Choice{}, fmt.Errorf("error running prompt: %w", err)
	}

	// Assert the final tea.Model to our local model and print the choice.
	if m, ok := m.(model); ok {
		return m.choice, nil
	}

	return Choice{}, fmt.Errorf("no choice made")
}

type Choice struct {
	Id           string
	Display      string
	OriginalData any
}

type model struct {
	choices []Choice
	cursor  int
	choice  Choice
	prompt  string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.choice = Choice{}
			return m, tea.Quit

		case "enter":
			// Send the choice on the channel and exit.
			m.choice = m.choices[m.cursor]
			return m, tea.Quit

		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.choices) {
				m.cursor = 0
			}

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.choices) - 1
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	s := strings.Builder{}
	s.WriteString(m.prompt)
	s.WriteString("\n")

	for i := 0; i < len(m.choices); i++ {
		if m.cursor == i {
			s.WriteString(" (•) ")
		} else {
			s.WriteString(" ( ) ")
		}
		s.WriteString(m.choices[i].Display)
		s.WriteString("\n")
	}

	return s.String()
}
