package models

type Member struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Email        string                `json:"email"`
	Organization OrganizationReference `json:"organization"`
	Rules        []Rule                `json:"rules"`
}
