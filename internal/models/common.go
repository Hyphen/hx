package models

type PaginatedResponse[T any] struct {
	Data     []T `json:"data"`
	PageNum  int `json:"pageNum,omitempty"`
	PageSize int `json:"pageSize,omitempty"`
	Total    int `json:"total,omitempty"`
}
