package code

import (
	"encoding/json"
	"fmt"

	"github.com/Hyphen/cli/internal/run"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type WebSocketMessage struct {
	EventStreamTopic string          `json:"eventStreamTopic"`
	OrganizationId   string          `json:"organizationId"`
	Data             json.RawMessage `json:"data"`
}

type RunData struct {
	Action string  `json:"action"`
	RunId  string  `json:"runId"`
	Run    run.Run `json:"run"`
}

type GenerateDockerRunModel struct {
	RunID string
	Run   *run.Run
}

var (
	spinIcon = spinner.New()
)

func (m GenerateDockerRunModel) Init() tea.Cmd {
	spinIcon.Spinner = spinner.Line
	return spinIcon.Tick
}

func (m GenerateDockerRunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		spinIcon, cmd = spinIcon.Update(msg)
		return m, cmd
	case RunData:
		m.Run.Status = msg.Run.Status
	}
	return m, nil
}

func (m GenerateDockerRunModel) View() string {
	result := "-------------------------------------------------\n"
	result += getMarkerBasedOnStatus(m.Run.Status) + "\n"
	result += "-------------------------------------------------\n"
	return result
}

func getMarkerBasedOnStatus(status run.RunStatus) string {
	switch status {
	case run.RunStatusPending:
		fallthrough
	case run.RunStatusQueued:
		fallthrough
	case run.RunStatusRunning:
		return fmt.Sprintf("%s Generating Dockerfile (this may take a few seconds)...", spinIcon.View())
	case run.RunStatusSucceeded:
		return "✅ Dockerfile generated! You may choose to check it in if you like."
	case run.RunStatusFailed:
		return "❌ Dockerfile generation failed!"
	case run.RunStatusCanceled:
		return "🚫 Dockerfile generation was canceled."
	default:
		return "❓ Unknown status"
	}
}
