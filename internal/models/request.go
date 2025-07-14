package models

type Request struct {
	ID            string `json:"id"`
	CausationID   string `json:"causationId"`
	CorrelationID string `json:"correlationId"`
}
