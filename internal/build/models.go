package build

import common "github.com/Hyphen/cli/internal"

type Artifact struct {
	Type  string `json:"type"`
	Image struct {
		URI string `json:"uri"`
	} `json:"image"`
	Ports []string `json:"ports"`
}

type Build struct {
	Id                 string                                         `json:"id"`
	Organization       common.OrganizationReference                   `json:"organization"`
	Project            common.ProjectReference                        `json:"project"`
	ProjectEnvironment common.ProjectEnvironmentWithWildCardReference `json:"projectEnvironment"`
	App                common.AppReference                            `json:"app"`
	Tags               []string                                       `json:"tags;omitempty"`
	CommitSha          string                                         `json:"commitSha"`
	Artifact           Artifact                                       `json:"artifact"`
}

type NewBuild struct {
	Tags      []string `json:"tags"`
	CommitSha string   `json:"commitSha"`
	Artifact  Artifact `json:"artifact"`
}
