package models

type Artifact struct {
	Type  string `json:"type"`
	Image struct {
		URI string `json:"uri"`
	} `json:"image"`
	Ports []int `json:"ports"`
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
	Tags      []string `json:"tags"`
	CommitSha string   `json:"commitSha"`
	Artifact  Artifact `json:"artifact"`
}
