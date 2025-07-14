package deployment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type IDeploymentService interface {
	searchDeployments(organizationId, nameOrId string, pageSize, pageNum int) ([]models.Deployment, error)
}

type DeploymentService struct {
	baseUrl    string
	httpClient httputil.Client
}

type AppSources struct {
	AppId    string          `json:"appId"`
	Artifact models.Artifact `json:"artifact,omitempty"`
	BuildId  string          `json:"buildId,omitempty"`
}

func NewService() *DeploymentService {
	baseUrl := apiconf.GetBaseApixUrl()
	return &DeploymentService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func (ds *DeploymentService) CreateRun(organizationId, deploymentId string, appSources []AppSources) (*models.DeploymentRun, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/deployments/%s/runs", ds.baseUrl, organizationId, deploymentId)
	//app_67af84d8cf5902a8f372bbcc
	//requestBody := []byte("{\"artifacts\":[{\"appId\":\"app_67af84d8cf5902a8f372bbcc\",\"image\":\"us-docker.pkg.dev/hyphenai/public/deploy-demo\"}]}")
	requestBody, _ := json.Marshal(map[string]interface{}{
		"artifacts": appSources,
	})

	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewReader(requestBody)))
	if err != nil {
		return nil, err
	}

	resp, err := ds.httpClient.Do(req)
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

	var deploymentRun models.DeploymentRun
	err = json.Unmarshal(body, &deploymentRun)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return &deploymentRun, nil
}

func (ds *DeploymentService) GetDeploymentRun(organizationId, deploymentId, runId string) (*models.DeploymentRun, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/deployments/%s/runs/%s", ds.baseUrl, organizationId, deploymentId, runId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ds.httpClient.Do(req)
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

	var deploymentRun models.DeploymentRun
	err = json.Unmarshal(body, &deploymentRun)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return &deploymentRun, nil
}

func (ds *DeploymentService) SearchDeployments(organizationId, nameOrId string, pageSize, pageNum int) ([]models.Deployment, error) {
	queryParams := url.Values{}
	queryParams.Set("pageNum", fmt.Sprintf("%d", pageNum))
	queryParams.Set("pageSize", fmt.Sprintf("%d", pageSize))
	queryParams.Set("search", nameOrId)

	url := fmt.Sprintf("%s/api/organizations/%s/deployments/?%s", ds.baseUrl, organizationId, queryParams.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ds.httpClient.Do(req)
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

	var response models.PaginatedResponse[models.Deployment]

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return response.Data, nil
}
