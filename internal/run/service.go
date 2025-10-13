package run

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type IRunService interface {
}

type RunService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService() *RunService {
	baseUrl := apiconf.GetBaseApixUrl()
	return &RunService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func (rs *RunService) CreateRun() {

}

func (rs *RunService) CreateDockerFileRun(organizationID, appID, targetBranch string) (*Run, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/runs", rs.baseUrl, organizationID, appID)

	requestBody, _ := json.Marshal(map[string]interface{}{
		"type":         RunTypeGenerateDockerfile,
		"targetBranch": targetBranch,
	})

	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewReader(requestBody)))
	if err != nil {
		return nil, err
	}

	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var run Run
	err = json.Unmarshal(body, &run)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return &run, nil
}
