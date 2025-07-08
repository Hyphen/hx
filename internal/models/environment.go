package models

type Environment struct {
	ID           string                `json:"id"`
	AlternateID  string                `json:"alternateId"`
	Name         string                `json:"name"`
	Color        string                `json:"color"`
	Organization OrganizationReference `json:"organization"`
	Project      ProjectReference      `json:"project"`
}
