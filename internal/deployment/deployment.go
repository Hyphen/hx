package Deployment

type Deployment struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Organization OrganizationReference `json:"organization"`
}

type OrganizationReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeploymentRun struct {
	ID                 string                `json:"id"`
	Status             string                `json:"status"`
	DeploymentId       string                `json:"deploymentId"`
	DeploymentSnapshot Deployment            `json:"deploymentSnapshot"`
	Organization       OrganizationReference `json:"organization"`
	Pipeline           Pipeline              `json:"pipeline"`
}

type Step struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	ParallelSteps []Step `json:"parallelSteps,omitempty"`
	Tasks         []Task `json:"tasks,omitempty"`
}

type Task struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

type Pipeline struct {
	Steps []Step `json:"steps"`
}
