package models

type Integration struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Repository struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"fullName"`
	Private  bool   `json:"private"`
}

type AppConnection struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Status       string             `json:"status"`
	Entity       ConnectionEntity   `json:"entity"`
	Repository   Repository         `json:"repository"`
	Organization OrganizationReference `json:"organization"`
}
