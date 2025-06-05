package common

type OrganizationReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectReference struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AlternativeId string `json:"alternativeId"`
}

type AppReference struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AlternativeId string `json:"alternativeId"`
}

type ProjectEnvironmentWithWildCardReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectEnvironmentReference struct {
	ID            string `json:"id"`
	AlternativeId string `json:"alternativeId"`
	Name          string `json:"name"`
}

type PaginatedResponse[T any] struct {
	Data     []T `json:"data"`
	PageNum  int `json:"pageNum,omitempty"`
	PageSize int `json:"pageSize,omitempty"`
	Total    int `json:"total,omitempty"`
}
