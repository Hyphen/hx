package projects

type Project struct {
	ID          *string `json:"id,omitempty"`
	AlternateID string  `json:"alternateId"`
	Name        string  `json:"name"`
}

type ProjectsListResponse struct {
	Data []Project `json:"data"`
}
