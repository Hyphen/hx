package models

type ProjectEnvironmentReference struct {
	ID          string `json:"id"`
	AlternateID string `json:"alternateId"`
	Name        string `json:"name"`
}

type ProjectEnvironmentWithWildCardReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
