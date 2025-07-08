package models

// TODO -- is there actually more to this?
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type OrganizationReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
