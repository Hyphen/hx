package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type ProjectServicer interface {
	GetListProjects(organizationID string, pageSize, pageNum int) ([]Project, error)
	CreateProject(organizationID, alternateID, name string) (Project, error)
	GetProject(organizationID, projectID string) (Project, error)
	DeleteProject(organizationID, projectID string) error
}

type ProjectService struct {
	baseUrl      string
	oauthService oauth.OAuthServicer
	httpClient   httputil.Client
}

func NewService() *ProjectService {
	baseUrl := "https://dev-api.hyphen.ai"
	if customAPI := os.Getenv("HYPHEN_CUSTOM_APIX"); customAPI != "" {
		baseUrl = customAPI
	}
	return &ProjectService{
		baseUrl:      baseUrl,
		oauthService: oauth.DefaultOAuthService(),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

func (ps *ProjectService) GetListProjects(organizationID string, pageSize, pageNum int) ([]Project, error) {
	token, err := ps.oauthService.GetValidToken()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/projects/?pageNum=%d&pageSize=%d",
		ps.baseUrl, organizationID, pageNum, pageSize)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var response struct {
		Total    int       `json:"total"`
		PageNum  int       `json:"pageNum"`
		PageSize int       `json:"pageSize"`
		Data     []Project `json:"data"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return response.Data, nil
}

func (ps *ProjectService) CreateProject(organizationID, alternateID, name string) (Project, error) {
	token, err := ps.oauthService.GetValidToken()
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/projects/", ps.baseUrl, organizationID)

	payload := struct {
		AlternateID string `json:"alternateId"`
		Name        string `json:"name"`
	}{
		AlternateID: alternateID,
		Name:        name,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to marshal request payload")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return Project{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to read response body")
	}

	var project Project
	err = json.Unmarshal(body, &project)
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return project, nil
}

func (ps *ProjectService) GetProject(organizationID, projectID string) (Project, error) {
	token, err := ps.oauthService.GetValidToken()
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/projects/%s/", ps.baseUrl, organizationID, projectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Project{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to read response body")
	}

	var project Project
	err = json.Unmarshal(body, &project)
	if err != nil {
		return Project{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return project, nil
}

func (ps *ProjectService) DeleteProject(organizationID, projectID string) error {
	token, err := ps.oauthService.GetValidToken()
	if err != nil {
		return errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/projects/%s/", ps.baseUrl, organizationID, projectID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errors.HandleHTTPError(resp)
	}

	return nil
}
