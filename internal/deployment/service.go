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
	CreateEnvironmentDeployment(organizationId, projectId, projectEnvironmentId, appId, name, alternateId, description string) (*models.Deployment, error)
}

type createEnvironmentDeploymentRequest struct {
	Name               string                 `json:"name"`
	AlternateID        string                 `json:"alternateId"`
	Description        string                 `json:"description"`
	Project            models.ProjectReference            `json:"project"`
	ProjectEnvironment models.ProjectEnvironmentReference `json:"projectEnvironment"`
	Apps               []createDeploymentApp  `json:"apps"`
}

type createDeploymentApp struct {
	App models.AppReference `json:"app"`
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

func (ds *DeploymentService) CreateRun(organizationId, deploymentId string, appSources []AppSources, previewId string) (*models.DeploymentRun, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/deployments/%s/runs", ds.baseUrl, organizationId, deploymentId)
	//app_67af84d8cf5902a8f372bbcc
	//requestBody := []byte("{\"artifacts\":[{\"appId\":\"app_67af84d8cf5902a8f372bbcc\",\"image\":\"us-docker.pkg.dev/hyphenai/public/deploy-demo\"}]}")
	requestPayload := map[string]interface{}{
		"artifacts": appSources,
	}
	if previewId != "" {
		requestPayload["previewId"] = previewId
	}
	requestBody, _ := json.Marshal(requestPayload)

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

func (ds *DeploymentService) CreatePreview(organizationId string, deployment models.Deployment, name string, hostPrefix string) (*models.DeploymentPreview, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/deployments/%s/previews/", ds.baseUrl, organizationId, deployment.ID)

	requestPayload := map[string]interface{}{
		"name":       name,
		"hostPrefix": hostPrefix,
	}
	requestBody, _ := json.Marshal(requestPayload)

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

	var preview models.DeploymentPreview
	err = json.Unmarshal(body, &preview)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return &preview, nil
}

func (ds *DeploymentService) CreateEnvironmentDeployment(organizationId, projectId, projectEnvironmentId, appId, name, alternateId, description string) (*models.Deployment, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/deployments/", ds.baseUrl, organizationId)

	requestBody, err := json.Marshal(createEnvironmentDeploymentRequest{
		Name:        name,
		AlternateID: alternateId,
		Description: description,
		Project:     models.ProjectReference{ID: projectId},
		ProjectEnvironment: models.ProjectEnvironmentReference{ID: projectEnvironmentId},
		Apps: []createDeploymentApp{
			{App: models.AppReference{ID: appId}},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal request body")
	}

	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewReader(requestBody)))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var deployment models.Deployment
	if err := json.Unmarshal(body, &deployment); err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return &deployment, nil
}
