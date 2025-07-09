package models

type App struct {
	ID           string                `json:"id"`
	AlternateId  string                `json:"alternateId"`
	Name         string                `json:"name"`
	Organization OrganizationReference `json:"organization"`
	Project      ProjectReference      `json:"project"`
}

type AppReference struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AlternativeId string `json:"alternativeId"`
}
