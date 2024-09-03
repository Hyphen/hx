package members

type Member struct {
	ID                string             `json:"id"`
	FirstName         string             `json:"firstName"`
	LastName          string             `json:"lastName"`
	Email             string             `json:"email"`
	ConnectedAccounts []ConnectedAccount `json:"connectedAccounts,omitempty"`
}

type ConnectedAccount struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
	ProfileURL string `json:"profileUrl"`
}
