package appcloud

// App mirrors the AppCloud management API `AppView` (camelCase JSON). Only the
// fields the CLI surfaces are modelled; unknown fields are ignored.
type App struct {
	ID             string    `json:"id"`
	Owner          string    `json:"owner"`
	Name           string    `json:"name"`
	OrganizationID string    `json:"organizationId"`
	ProjectID      string    `json:"projectId"`
	Domains        []string  `json:"domains"`
	ActiveRevision string    `json:"activeRevision"`
	Config         AppConfig `json:"config"`
	ConfigETag     string    `json:"configEtag"`
	CreatedAt      string    `json:"createdAt"`
	UpdatedAt      string    `json:"updatedAt"`
}

// AppConfig is the app's routing/serving config as returned by the API. It is
// intentionally permissive: the management API owns the schema, so we keep the
// raw representation and only pull out what the CLI needs.
type AppConfig map[string]any

// listResponse is the shape of `GET /v1/apps` (items + optional cursor).
type listResponse struct {
	Items      []App  `json:"items"`
	NextCursor string `json:"nextCursor"`
}
