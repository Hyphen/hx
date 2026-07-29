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

// AppConfig is the app's routing/serving config. The management API owns the
// schema; the CLI keeps the raw representation and round-trips it verbatim.
type AppConfig map[string]any

// Revision mirrors the management `RevisionView`.
type Revision struct {
	ID          string `json:"id"`
	AppID       string `json:"appId"`
	Hex         string `json:"hex"`
	Kind        string `json:"kind"`
	ArtifactRef string `json:"artifactRef"`
	PreviewURL  string `json:"previewUrl"`
	CreatedAt   string `json:"createdAt"`
}

// ConfigView is the body of `GET`/`PUT /v1/apps/{id}/config`.
type ConfigView struct {
	Config     AppConfig `json:"config"`
	ConfigETag string    `json:"configEtag"`
}

// UploadResponse is the body of a revision file-upload batch.
type UploadResponse struct {
	Files []UploadedFile `json:"files"`
}

type UploadedFile struct {
	Path string `json:"path"`
	Key  string `json:"key"`
	Size uint64 `json:"size"`
}

// HTTPLog mirrors management's `HttpLogView` (one served request).
type HTTPLog struct {
	ID         string `json:"id"`
	TS         string `json:"ts"`
	Domain     string `json:"domain"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	Bytes      uint64 `json:"bytes"`
	DurationMS uint64 `json:"durationMs"`
	Revision   string `json:"revision"`
	CacheHit   bool   `json:"cacheHit"`
}

// ErrorLog mirrors management's `ErrorLogView`.
type ErrorLog struct {
	ID       string `json:"id"`
	TS       string `json:"ts"`
	Domain   string `json:"domain"`
	Kind     string `json:"kind"`
	Message  string `json:"message"`
	Revision string `json:"revision"`
}

// listResponse is the generic `{items, nextCursor}` envelope.
type listResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor"`
}
