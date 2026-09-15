package models

type Artifact struct {
	Type  string `json:"type"`
	Image *struct {
		URI string `json:"uri"`
	} `json:"image,omitempty"`
	Ports  []int           `json:"ports,omitzero"`
	Target string          `json:"target,omitempty"`
	Site   *StaticArtifact `json:"site,omitempty"`
}

type StaticArtifact struct {
	RegistryID string `json:"registryId"`
	Revision   string `json:"revision"`
	URI        string `json:"uri,omitempty"`
}

type Build struct {
	Id                 string                                  `json:"id"`
	Organization       OrganizationReference                   `json:"organization"`
	Project            ProjectReference                        `json:"project"`
	ProjectEnvironment ProjectEnvironmentWithWildCardReference `json:"projectEnvironment"`
	App                AppReference                            `json:"app"`
	Tags               []string                                `json:"tags;omitempty"`
	CommitSha          string                                  `json:"commitSha"`
	Artifact           Artifact                                `json:"artifact"`
}

type NewBuild struct {
	Tags          []string `json:"tags"`
	CommitSha     string   `json:"commitSha"`
	CommitShaHref string   `json:"commitShaHref,omitempty"`
	Tag           string   `json:"tag,omitempty"`
	TagHref       string   `json:"tagHref,omitempty"`
	Artifact      Artifact `json:"artifact"`
}
