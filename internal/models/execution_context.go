package models

type ExecutionContext struct {
	Request Request `json:"request"`
	User    User    `json:"user"`
	Member  Member  `json:"member"`
}
