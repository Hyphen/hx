package projects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type ProjectServicer interface {
	ListProjects() ([]models.Project, error)
	GetProject(projectID string) (models.Project, error)
	CreateProject(project models.Project) (models.Project, error)
}

type ProjectService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService(organizationID string) ProjectService {
	baseUrl := fmt.Sprintf("%s/api/organizations/%s/projects", apiconf.GetBaseApixUrl(), organizationID)
	return ProjectService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func NewTestService(organizationID string, httpClient httputil.Client) ProjectServicer {
	baseUrl := fmt.Sprintf("%s/api/organizations/%s/projects", apiconf.GetBaseApixUrl(), organizationID)
	return &ProjectService{
		baseUrl,
		httpClient,
	}
}

func (ps *ProjectService) ListProjects() ([]models.Project, error) {
	url := fmt.Sprintf("%s/", ps.baseUrl)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []models.Project{}, errors.Wrap(err, "Failed to create request")
	}
	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return []models.Project{}, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []models.Project{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []models.Project{}, errors.Wrap(err, "Failed to read response body")
	}

	type projectsListResponse struct {
		Data       []models.Project `json:"data"`
		TotalCount int              `json:"totalCount"`
		PageNum    int              `json:"pageNum"`
		PageSize   int              `json:"pageSize"`
	}

	// unmarshal the body
	var projectsResponse projectsListResponse
	err = json.Unmarshal(body, &projectsResponse)
	if err != nil {
		return []models.Project{}, errors.Wrap(err, "Failed to unmarshal response body")
	}

	return projectsResponse.Data, nil
}

func (ps *ProjectService) GetProject(projectID string) (models.Project, error) {
	url := fmt.Sprintf("%s/%s", ps.baseUrl, projectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return models.Project{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return models.Project{}, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Project{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return models.Project{}, errors.Wrap(err, "Failed to read response body")
	}

	var project models.Project
	err = json.Unmarshal(body, &project)
	if err != nil {
		return models.Project{}, errors.Wrap(err, "Failed to unmarshal response body")
	}

	return project, nil
}

func (ps *ProjectService) CreateProject(project models.Project) (models.Project, error) {
	url := fmt.Sprintf("%s/", ps.baseUrl)

	body, err := json.Marshal(project)
	if err != nil {
		return models.Project{}, errors.Wrap(err, "Failed to marshal project")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return models.Project{}, errors.Wrap(err, "Failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return models.Project{}, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return models.Project{}, errors.HandleHTTPError(resp)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return models.Project{}, errors.Wrap(err, "Failed to read response body")
	}

	var createdProject models.Project
	err = json.Unmarshal(body, &createdProject)
	if err != nil {
		return models.Project{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return createdProject, nil
}

func CheckProjectId(appId string) error {
	validRegex := regexp.MustCompile("^[a-z0-9-_]+$")
	if !validRegex.MatchString(appId) {
		suggested := strings.ToLower(appId)
		suggested = strings.ReplaceAll(suggested, " ", "-")
		suggested = regexp.MustCompile("[^a-z0-9-_]").ReplaceAllString(suggested, "-")
		suggested = regexp.MustCompile("-+").ReplaceAllString(suggested, "-")
		suggested = strings.Trim(suggested, "-")

		return errors.Wrapf(
			errors.New("invalid project ID"),
			"You are using unpermitted characters. A valid Project ID can only contain lowercase letters, numbers, hyphens, and underscores. Suggested valid ID: %s",
			suggested,
		)
	}

	return nil
}
