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
