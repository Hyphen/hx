package models

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
	Type  string `json:"type"`
}
