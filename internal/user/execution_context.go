package user

type Request struct {
	ID            string `json:"id"`
	CausationID   string `json:"causationId"`
	CorrelationID string `json:"correlationId"`
}

type Condition struct {
	ConnectedAccounts *ConnectedAccounts `json:"connectedAccounts,omitempty"`
	ID                string             `json:"id,omitempty"`
	OrganizationID    string             `json:"organization.id,omitempty"`
}

type ConnectedAccounts struct {
	ElemMatch *ElemMatch `json:"$elemMatch"`
}

type ElemMatch struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

type Rule struct {
	Action     string     `json:"action"`
	Subject    string     `json:"subject"`
	Conditions *Condition `json:"conditions,omitempty"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
	Type  string `json:"type"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Member struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Organization Organization `json:"organization"`
	Rules        []Rule       `json:"rules"`
}

type ExecutionContext struct {
	Request Request `json:"request"`
	User    User    `json:"user"`
	Member  Member  `json:"member"`
}
