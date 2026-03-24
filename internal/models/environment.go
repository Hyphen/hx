package models

type EnvironmentType string

const (
	EnvironmentTypeDevelopment EnvironmentType = "development"
	EnvironmentTypeProduction  EnvironmentType = "production"
	EnvironmentTypeCustom      EnvironmentType = "custom"
)


type Environment struct {
	ID           string                `json:"id"`
	AlternateID  string                `json:"alternateId"`
	Name         string                `json:"name"`
	Color        string                `json:"color"`
	Organization OrganizationReference `json:"organization"`
	Project      ProjectReference      `json:"project"`
	Type         EnvironmentType       `json:"type"`
}
