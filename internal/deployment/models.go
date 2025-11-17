package deployment

import (
	"encoding/json"
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
	Type     string                    `json:"type"`
	RunId    string                    `json:"RunId"`
	Id       string                    `json:"id"`
	Status   string                    `json:"status"`
	Pipeline *models.DeploymentPipeline `json:"pipeline,omitempty"`
}

type StatusModel struct {
	Pipeline       models.DeploymentPipeline
	OrganizationId string
	DeploymentId   string
	RunId          string
	Service        DeploymentService
	AppUrl         string
}

var (
	spinIcon = spinner.New()
)

func (m *StatusModel) Init() tea.Cmd {
	spinIcon.Spinner = spinner.Line
	return spinIcon.Tick
}

func (m *StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var spinCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		spinIcon, spinCmd = spinIcon.Update(msg)
		return m, spinCmd
	case RunMessageData:
		switch msg.Type {
		case "run":
			if msg.Pipeline != nil {
				// Update the pipeline with the data sent in the message
				m.Pipeline = *msg.Pipeline
			}
			m.UpdateStatusForId(msg.Id, msg.Status)
		case "step":
			m.UpdateStatusForId(msg.Id, msg.Status)
		case "task":
			m.UpdateStatusForId(msg.Id, msg.Status)
		}
		// Always keep the spinner ticking
		return m, spinIcon.Tick
	}

	return m, spinIcon.Tick
}

func (m *StatusModel) View() string {
	result := "-------------------------------------------------\n"
	result += m.AppUrl + "\n"
	result += "-------------------------------------------------\n"
	result += m.RenderTree(m.Pipeline)
	return result
}

func (m *StatusModel) RenderTree(pipeLine models.DeploymentPipeline) string {
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
	const (
		green = "\033[32m"
		red   = "\033[31m"
		amber = "\033[33m"
		reset = "\033[0m"
	)

	switch status {
	case "succeeded", "Success", "Succeeded":
		return "[" + green + "✓" + reset + "]"
	case "failed", "Error", "canceled", "Failed", "Canceled":
		return "[" + red + "✗" + reset + "]"
	case "running", "Running":
		return "[" + amber + spinIcon.View() + reset + "]"
	case "pending", "queued", "Pending", "Queued":
		return "[ ]"
	default:
		return "[ ]" // Default to empty checkbox for unknown statuses
	}
}

func (m *StatusModel) UpdateStatusForId(id string, status string) {
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
