package build

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type BuildService struct {
	baseUrl    string
	httpClinet httputil.Client
}

func NewService() *BuildService {
	baseUrl := apiconf.GetBaseApixUrl()
	return &BuildService{
		baseUrl:    baseUrl,
		httpClinet: httputil.NewHyphenHTTPClient(),
	}
}

func (bs *BuildService) CreateBuild(organizationId, appId, commitSha, dockerUri string) (*Build, error) {
	///api/organizations/{organizationId}/apps/{appId}/builds/
	url := bs.baseUrl + "/api/organizations/" + organizationId + "/apps/" + appId + "/builds"

	build := NewBuild{
		Tags:      []string{},
		CommitSha: commitSha,
		Artifact: Artifact{
			Type: "Docker",
			Image: struct {
				URI string `json:"uri"`
			}{
				URI: dockerUri,
			},
		},
	}

	buildJSON, err := json.Marshal(build)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal build data to JSON")
	}

	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewBuffer(buildJSON)))
	if err != nil {
		return nil, err
	}

	resp, err := bs.httpClinet.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var NewBuild Build
	err = json.Unmarshal(body, &NewBuild)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}
	return &NewBuild, nil
}
