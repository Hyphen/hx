package models

type Condition struct {
	ConnectedAccounts *ConnectedAccounts `json:"connectedAccounts,omitempty"`
	ID                string             `json:"id,omitempty"`
	OrganizationID    string             `json:"organization.id,omitempty"`
}
