package app

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

type AppServicer interface {
	GetListApps(organizationID, projectID string, pageSize, pageNum int) ([]models.App, error)
	CreateApp(organizationID, projectID, alternateID, name string) (models.App, error)
	GetApp(organizationID, appID string) (models.App, error)
	DeleteApp(organizationID, appID string) error
}

type AppService struct {
	baseUrl    string
	httpClient httputil.Client
}

func NewService() *AppService {
	baseUrl := apiconf.GetBaseApixUrl()
	return &AppService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

func (ps *AppService) GetListApps(organizationID, projectID string, pageSize, pageNum int) ([]models.App, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/?pageNum=%d&pageSize=%d&projects=%s", ps.baseUrl, organizationID, pageNum, pageSize, projectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create request")
	}

	resp, err := ps.httpClient.Do(req)
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

	var response models.PaginatedResponse[models.App]

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return response.Data, nil
}

func (ps *AppService) CreateApp(organizationID, projectId, alternateID, name string) (models.App, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/projects/%s/apps", ps.baseUrl, organizationID, projectId)

	payload := struct {
		AlternateID string `json:"alternateId"`
		Name        string `json:"name"`
	}{
		AlternateID: alternateID,
		Name:        name,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return models.App{}, errors.Wrap(err, "Failed to marshal request payload")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return models.App{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return models.App{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return models.App{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.App{}, errors.Wrap(err, "Failed to read response body")
	}

	var app models.App
	err = json.Unmarshal(body, &app)
	if err != nil {
		return models.App{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return app, nil
}

func (ps *AppService) GetApp(organizationID, appID string) (models.App, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/", ps.baseUrl, organizationID, appID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return models.App{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return models.App{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.App{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.App{}, errors.Wrap(err, "Failed to read response body")
	}

	var app models.App
	err = json.Unmarshal(body, &app)
	if err != nil {
		return models.App{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return app, nil
}

func (ps *AppService) DeleteApp(organizationID, appID string) error {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/", ps.baseUrl, organizationID, appID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create request")
	}

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errors.HandleHTTPError(resp)
	}

	return nil
}

func CheckAppId(appId string) error {
	validRegex := regexp.MustCompile("^[a-z0-9-_]+$")
	if !validRegex.MatchString(appId) {
		suggested := strings.ToLower(appId)
		suggested = strings.ReplaceAll(suggested, " ", "-")
		suggested = regexp.MustCompile("[^a-z0-9-_]").ReplaceAllString(suggested, "-")
		suggested = regexp.MustCompile("-+").ReplaceAllString(suggested, "-")
		suggested = strings.Trim(suggested, "-")

		return errors.Wrapf(
			errors.New("invalid app ID"),
			"You are using unpermitted characters. A valid App ID can only contain lowercase letters, numbers, hyphens, and underscores. Suggested valid ID: %s",
			suggested,
		)
	}

	return nil
}
