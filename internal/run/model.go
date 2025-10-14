package run

import "encoding/json"

type OrganizationReference struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Run struct {
	ID           string                 `json:"id,omitempty"`
	Type         RunType                `json:"type,omitempty"`
	Status       RunStatus              `json:"status,omitempty"`
	Organization *OrganizationReference `json:"organization,omitempty"`
	WorkflowId   string                 `json:"workflowId,omitempty"`
	Data         json.RawMessage        `json:"data,omitempty"`
	Input        interface{}            `json:"input,omitempty"`
}

type CodeChangeRunData struct {
	Changes []DiffResult `json:"changes,omitempty"`
}

type DiffResult struct {
	Chunks []struct {
		Content string `json:"content"`
		Type    string `json:"type"`
	} `json:"chunks"`
	From     string `json:"from"`
	To       string `json:"to"`
	FromMode string `json:"fromMode"`
	ToMode   string `json:"toMode"`
}

type RunType string

const (
	RunTypeDeployment         RunType = "deployment"
	RunTypeGenerateDockerfile RunType = "generateDockerfile"
	RunTypeResourceCleanup    RunType = "resourceCleanup"
)

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusQueued    RunStatus = "queued"
	RunStatusRunning   RunStatus = "running"
	RunStatusSucceeded RunStatus = "succeeded"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCanceled  RunStatus = "canceled"
)
