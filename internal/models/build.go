package models

import "encoding/json"

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
	Provider   string `json:"-"`
	RegistryID string `json:"registryId"`
	URI        string `json:"uri"`

	Options json.RawMessage `json:"options,omitempty"`
}

type Build struct {
	Id                 string                                  `json:"id"`
	Organization       OrganizationReference                   `json:"organization"`
	Project            ProjectReference                        `json:"project"`
	ProjectEnvironment ProjectEnvironmentWithWildCardReference `json:"projectEnvironment"`
	App                AppReference                            `json:"app"`
	Tags               []string                                `json:"tags,omitempty"`
	CommitSha          string                                  `json:"commitSha"`
	Artifacts          []Artifact                              `json:"artifacts"`
}

type NewBuild struct {
	Tags          []string   `json:"tags"`
	CommitSha     string     `json:"commitSha"`
	CommitShaHref string     `json:"commitShaHref,omitempty"`
	Tag           string     `json:"tag,omitempty"`
	TagHref       string     `json:"tagHref,omitempty"`
	Artifacts     []Artifact `json:"artifacts"`
}
