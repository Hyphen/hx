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
	Pipeline           DeploymentPipeline    `json:"pipeline"`
}

type DeploymentStep struct {
	ID            string           `json:"id"`
	Status        string           `json:"status"`
	Type          string           `json:"type"`
	Name          string           `json:"name"`
	ParallelSteps []DeploymentStep `json:"parallelSteps,omitempty"`
	Tasks         []DeploymentTask `json:"tasks,omitempty"`
}

type DeploymentTask struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

type DeploymentPipeline struct {
	Steps []DeploymentStep `json:"steps"`
}
