package models

type Rule struct {
	Action     string     `json:"action"`
	Subject    string     `json:"subject"`
	Conditions *Condition `json:"conditions,omitempty"`
}
