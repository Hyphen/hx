package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type IntegrationsServicer interface {
	ListIntegrations(organizationID string) ([]models.Integration, error)
	ListRepositories(organizationID, integrationID string) ([]models.Repository, error)
	ConnectAppToRepository(organizationID, appID, integrationID, repositoryName string, isNew bool) (models.AppConnection, error)
}

type IntegrationsService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService() *IntegrationsService {
	baseUrl := apiconf.GetBaseApixUrl()
	return &IntegrationsService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func (s *IntegrationsService) ListIntegrations(organizationID string) ([]models.Integration, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/integrations/", s.baseUrl, organizationID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create request")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var response models.PaginatedResponse[models.Integration]
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return response.Data, nil
}

func (s *IntegrationsService) ListRepositories(organizationID, integrationID string) ([]models.Repository, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/integrations/%s/repositories/", s.baseUrl, organizationID, integrationID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create request")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}

	var response models.PaginatedResponse[models.Repository]
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return response.Data, nil
}

func (s *IntegrationsService) ConnectAppToRepository(organizationID, appID, integrationID, repositoryName string, isNew bool) (models.AppConnection, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/connections", s.baseUrl, organizationID, appID)

	payload := struct {
		IntegrationID  string `json:"integrationId"`
		RepositoryName string `json:"repositoryName,omitempty"`
		IsNew          bool   `json:"isNew"`
	}{
		IntegrationID:  integrationID,
		RepositoryName: repositoryName,
		IsNew:          isNew,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return models.AppConnection{}, errors.Wrap(err, "Failed to marshal request payload")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return models.AppConnection{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return models.AppConnection{}, errors.Wrap(err, "Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return models.AppConnection{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.AppConnection{}, errors.Wrap(err, "Failed to read response body")
	}

	var connection models.AppConnection
	if err := json.Unmarshal(body, &connection); err != nil {
		return models.AppConnection{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return connection, nil
}
