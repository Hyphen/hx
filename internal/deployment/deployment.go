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
}
