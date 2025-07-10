package models

type ConnectedAccounts struct {
	ElemMatch *ElemMatch `json:"$elemMatch"`
}

type ElemMatch struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}
