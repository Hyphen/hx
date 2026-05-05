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
	SearchDeployments(organizationId, nameOrId string, pageSize, pageNum int, projectIds []string) ([]models.Deployment, error)
	CreateEnvironmentDeployment(organizationId, projectId, projectEnvironmentId, appId, name, alternateId, description string) (*models.Deployment, error)
	GetDeployment(organizationId, deploymentId string) (*models.Deployment, error)
}

type createEnvironmentDeploymentRequest struct {
	Name               string                             `json:"name"`
	AlternateID        string                             `json:"alternateId"`
	Description        string                             `json:"description"`
	Project            models.ProjectReference            `json:"project"`
	ProjectEnvironment models.ProjectEnvironmentReference `json:"projectEnvironment"`
	Apps               []createDeploymentApp              `json:"apps"`
}

type createDeploymentApp struct {
	App models.AppReference `json:"app"`
}

type DeploymentService struct {
	baseUrl    string
	httpClient httputil.Client
}

type AppSources struct {
	AppId    string           `json:"appId"`
	Artifact *models.Artifact `json:"artifact,omitempty"`
	BuildId  string           `json:"buildId,omitempty"`
	Build    string           `json:"build,omitempty"` // "latest" | "lastDeployed" | "latestPreview"
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

func (ds *DeploymentService) SearchDeployments(organizationId, nameOrId string, pageSize, pageNum int, projectIds []string) ([]models.Deployment, error) {
	queryParams := url.Values{}
	queryParams.Set("pageNum", fmt.Sprintf("%d", pageNum))
	queryParams.Set("pageSize", fmt.Sprintf("%d", pageSize))
	queryParams.Set("search", nameOrId)
	for _, id := range projectIds {
		queryParams.Add("projects", id)
	}

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

func (ds *DeploymentService) GetDeployment(organizationId, deploymentId string) (*models.Deployment, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/deployments/%s", ds.baseUrl, organizationId, deploymentId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

// patchDeploymentApp mirrors the subset of fields that PATCH
// /deployments/:id accepts on each app entry. Decoding the GET response into
// this shape naturally drops fields the PATCH validator rejects — notably
// projectEnvironment, which is part of the GET response but not part of the
// PATCH input schema (additionalProperties: false).
type patchDeploymentApp struct {
	Project            *patchProjectRef         `json:"project,omitempty"`
	App                patchAppRef              `json:"app"`
	DeploymentSettings *patchDeploymentSettings `json:"deploymentSettings,omitempty"`
}

type patchProjectRef struct {
	ID          string `json:"id"`
	AlternateID string `json:"alternateId,omitempty"`
	Name        string `json:"name,omitempty"`
}

type patchAppRef struct {
	ID          string `json:"id"`
	AlternateID string `json:"alternateId,omitempty"`
	Name        string `json:"name,omitempty"`
}

type patchDeploymentSettings struct {
	Targets        []patchTarget   `json:"targets"`
	Path           string          `json:"path,omitempty"`
	Hostname       string          `json:"hostname,omitempty"`
	DNS            *patchDNS       `json:"dns,omitempty"`
	Availability   string          `json:"availability"`
	Scale          string          `json:"scale"`
	TrafficRegions []string        `json:"trafficRegions"`
	Advanced       json.RawMessage `json:"advanced,omitempty"`
}

type patchTarget struct {
	ID   string `json:"id"`
	Type string `json:"type,omitempty"`
}

type patchDNS struct {
	ZoneName string `json:"zoneName"`
}

// AddAppsToDeployment patches a deployment to include the given app ids. Each
// new entry is sent without deploymentSettings, so the API fills in defaults
// (availability/scale/trafficRegions and the org's first cloud integration as
// the target). The PATCH replaces the apps array wholesale, so existing apps
// are re-sent verbatim from the GET response.
func (ds *DeploymentService) AddAppsToDeployment(organizationId, deploymentId string, appIds []string) (*models.Deployment, error) {
	if len(appIds) == 0 {
		return ds.GetDeployment(organizationId, deploymentId)
	}

	deploymentUrl := fmt.Sprintf(
		"%s/api/organizations/%s/deployments/%s",
		ds.baseUrl,
		url.PathEscape(organizationId),
		url.PathEscape(deploymentId),
	)

	getReq, err := http.NewRequest("GET", deploymentUrl, nil)
	if err != nil {
		return nil, err
	}
	getResp, err := ds.httpClient.Do(getReq)
	if err != nil {
		return nil, err
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		return nil, errors.HandleHTTPError(getResp)
	}

	var current struct {
		Apps []patchDeploymentApp `json:"apps"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&current); err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	// Apps without DNS come back as `dns: {}` in the GET response (the GET
	// schema only exposes zoneName). PATCH requires zoneName when dns is
	// present, so drop the empty DNS object before re-sending. Same for an
	// `advanced: null` from the wire — the PATCH schema requires an object.
	for i := range current.Apps {
		s := current.Apps[i].DeploymentSettings
		if s == nil {
			continue
		}
		if s.DNS != nil && s.DNS.ZoneName == "" {
			s.DNS = nil
		}
		if string(s.Advanced) == "null" {
			s.Advanced = nil
		}
	}

	for _, appId := range appIds {
		current.Apps = append(current.Apps, patchDeploymentApp{
			App: patchAppRef{ID: appId},
		})
	}

	requestBody, err := json.Marshal(map[string]any{
		"apps": current.Apps,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal request body")
	}

	req, err := http.NewRequest("PATCH", deploymentUrl, io.NopCloser(bytes.NewReader(requestBody)))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.HandleHTTPError(resp)
	}

	// The PATCH response doesn't include readiness fields (isReady,
	// readinessIssues) — only GET does. Re-fetch so callers get a fully
	// populated deployment.
	return ds.GetDeployment(organizationId, deploymentId)
}

func (ds *DeploymentService) CreateEnvironmentDeployment(organizationId, projectId, projectEnvironmentId, appId, name, alternateId, description string) (*models.Deployment, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/deployments/", ds.baseUrl, organizationId)

	requestBody, err := json.Marshal(createEnvironmentDeploymentRequest{
		Name:               name,
		AlternateID:        alternateId,
		Description:        description,
		Project:            models.ProjectReference{ID: projectId},
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
