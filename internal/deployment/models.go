package deployment

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Hyphen/cli/internal/models"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type WebSocketMessage struct {
	EventStreamTopic string          `json:"eventStreamTopic"`
	OrganizationId   string          `json:"organizationId"`
	Data             json.RawMessage `json:"data"`
}

// Define the first data type (current structure)
type LogMessageData struct {
	Level        string   `json:"level"`
	Message      string   `json:"message"`
	RunId        string   `json:"runId"`
	Timestamp    int64    `json:"timestamp"`
	Id           string   `json:"id"`
	Parents      []string `json:"parents"`
	Organization struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"organization"`
}

// Define the second data type (new structure)
type RunMessageData struct {
	Type   string `json:"type"`
	RunId  string `json:"runId"`
	Id     string `json:"id"`
	Status string `json:"status"`
}

type ErrorMessage struct {
	Error error
}

type WaitingTickMessage struct{}

type VerboseMessage struct {
	Content string
}

type StatusModel struct {
	Pipeline        models.DeploymentPipeline
	OrganizationId  string
	DeploymentId    string
	RunId           string
	Service         DeploymentService
	AppUrl          string
	Error           error
	WaitingSeconds  int
	VerboseMode     bool
	VerboseMessages []string
}

var (
	spinIcon = spinner.New()
)

func (m StatusModel) Init() tea.Cmd {
	spinIcon.Spinner = spinner.Line
	return spinIcon.Tick
}

func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case WaitingTickMessage:
		m.WaitingSeconds++
		return m, nil
	case VerboseMessage:
		m.VerboseMessages = append(m.VerboseMessages, msg.Content)
		return m, nil
	case ErrorMessage:
		m.Error = msg.Error
		return m, tea.Quit
	case RunMessageData:
		// Fetch pipeline if we don't have it yet (on any message type)
		if len(m.Pipeline.Steps) == 0 {
			run, _ := m.Service.GetDeploymentRun(m.OrganizationId, m.DeploymentId, m.RunId)
			if run != nil {
				m.Pipeline = run.Pipeline
			}
		}
		// Only update status if we have a pipeline
		if len(m.Pipeline.Steps) > 0 {
			m.UpdateStatusForId(msg.Id, msg.Status)
		}
	}

	return m, nil
}

func (m StatusModel) View() string {
	result := "-------------------------------------------------\n"
	result += m.AppUrl + "\n"
	result += "-------------------------------------------------\n"

	if len(m.Pipeline.Steps) == 0 {
		if m.VerboseMode {
			result += fmt.Sprintf("%s Waiting for deployment status... (%ds)\n", spinIcon.View(), m.WaitingSeconds)
		} else {
			result += fmt.Sprintf("%s Waiting for deployment status...\n", spinIcon.View())
		}
	}

	if m.VerboseMode && len(m.VerboseMessages) > 0 {
		result += "\nVerbose messages:\n"
		for _, msg := range m.VerboseMessages {
			result += fmt.Sprintf("  %s\n", msg)
		}
		result += "\n"
	}

	result += m.RenderTree(m.Pipeline)

	if m.Error != nil {
		result += fmt.Sprintf("❗error: %v\n", m.Error)
	}

	return result
}

func (m StatusModel) RenderTree(pipeLine models.DeploymentPipeline) string {
	var buildTree func(steps []models.DeploymentStep, level int) string

	buildTree = func(steps []models.DeploymentStep, level int) string {
		var result string
		indent := strings.Repeat("  ", level) // Indentation based on level

		for _, step := range steps {
			result += indent + getMarkerBasedOnStatus(step.Status) + " Step: " + step.Name + "\n"

			// Recursively handle parallel steps
			if len(step.ParallelSteps) > 0 {
				result += buildTree(step.ParallelSteps, level+2)
			}

			// Handle tasks
			for _, task := range step.Tasks {
				result += indent + "  " + getMarkerBasedOnStatus(task.Status) + " Task: " + task.Type + "\n"
			}
		}

		return result
	}

	return buildTree(pipeLine.Steps, 0)
}

func getMarkerBasedOnStatus(status string) string {
	switch status {
	case "Success":
		return "[✓]"
	case "Error":
		return "[✗]"
	case "Running":
		return fmt.Sprintf("[%s]", spinIcon.View())
	default:
		return "[ ]" // Default marker for unknown status
	}
}

func (m StatusModel) UpdateStatusForId(id string, status string) {
	// Helper function to recursively search and update status
	var updateStatus func(steps []models.DeploymentStep) bool
	updateStatus = func(steps []models.DeploymentStep) bool {
		for i := range steps {
			// Check if the current step matches the ID
			if steps[i].ID == id {
				steps[i].Status = status
				return true
			}

			// Check tasks within the step
			for j := range steps[i].Tasks {
				if steps[i].Tasks[j].ID == id {
					steps[i].Tasks[j].Status = status
					return true
				}
			}

			// Recursively search in parallel steps
			if updateStatus(steps[i].ParallelSteps) {
				return true
			}
		}
		return false
	}

	// Start searching from the top-level steps
	updateStatus(m.Pipeline.Steps)
}
