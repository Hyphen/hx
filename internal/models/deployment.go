package models

type DeploymentAppSettings struct {
	ProjectEnvironment ProjectEnvironmentReference `json:"projectEnvironment"`
	Scale              string                      `json:"scale"`
	Hostname           string                      `json:"hostname"`
	Availability       string                      `json:"availability"`
	Path               string                      `json:"path"`
}

type DeploymentApp struct {
	Project            ProjectReference      `json:"project"`
	App                AppReference          `json:"app"`
	DeploymentSettings DeploymentAppSettings `json:"deploymentSettings"`
}

type Deployment struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Organization OrganizationReference `json:"organization"`
	Apps         []DeploymentApp       `json:"apps"`
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
