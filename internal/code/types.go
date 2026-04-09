package code

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type StatusMessage struct {
	Content string
}

type SuccessMessage struct {
	Summary      string
	FilesWritten bool
}

type ErrorMessage struct {
	Error error
}

type VerboseMessage struct {
	Content string
}

type GenerateDockerSessionModel struct {
	Status          string
	Summary         string
	FilesWritten    bool
	VerboseMode     bool
	VerboseMessages []string
	Error           error
	Done            bool
}

var (
	spinIcon = spinner.New()
)

func (m GenerateDockerSessionModel) Init() tea.Cmd {
	spinIcon.Spinner = spinner.Line
	return spinIcon.Tick
}

func (m GenerateDockerSessionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		if m.Done || m.Error != nil {
			return m, nil
		}
		var cmd tea.Cmd
		spinIcon, cmd = spinIcon.Update(msg)
		return m, cmd
	case VerboseMessage:
		m.VerboseMessages = append(m.VerboseMessages, msg.Content)
		return m, nil
	case StatusMessage:
		m.Status = msg.Content
		return m, nil
	case SuccessMessage:
		m.Done = true
		m.Summary = msg.Summary
		m.FilesWritten = msg.FilesWritten
		return m, tea.Quit
	case ErrorMessage:
		m.Error = msg.Error
		return m, tea.Quit
	}

	return m, nil
}

func (m GenerateDockerSessionModel) View() string {
	result := "-------------------------------------------------\n"

	if m.VerboseMode && len(m.VerboseMessages) > 0 {
		for _, msg := range m.VerboseMessages {
			result += fmt.Sprintf("  %s\n", msg)
		}
		result += "\n"
	}

	switch {
	case m.Error != nil:
		result += "❌ Dockerfile generation failed!\n"
	case m.Done:
		if m.FilesWritten {
			result += "✅ Dockerfile and .dockerignore generated! You may choose to check them in if you like.\n"
		} else {
			result += "✅ Existing Dockerfile and .dockerignore already look good. No files were generated or changed.\n"
		}
	default:
		result += fmt.Sprintf("%s %s\n", spinIcon.View(), m.Status)
	}

	if m.Summary != "" {
		result += fmt.Sprintf("\n%s\n", m.Summary)
	}

	if m.Error != nil {
		result += fmt.Sprintf("\n❗error: %v\n", m.Error)
	}

	result += "-------------------------------------------------\n"
	return result
}
