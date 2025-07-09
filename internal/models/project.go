package models

type Project struct {
	ID          *string `json:"id,omitempty"`
	AlternateID string  `json:"alternateId"`
	Name        string  `json:"name"`
	IsMonorepo  bool    `json:"isMonorepo"`
}

type ProjectReference struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AlternateID string `json:"alternateId"`
}
