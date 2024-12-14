package app

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

type AppServicer interface {
	GetListApps(organizationID, projectID string, pageSize, pageNum int) ([]App, error)
	CreateApp(organizationID, projectID, alternateID, name string, isMonorepo bool) (App, error)
	GetApp(organizationID, appID string) (App, error)
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

func (ps *AppService) GetListApps(organizationID, projectID string, pageSize, pageNum int) ([]App, error) {
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

	var response struct {
		Total    int   `json:"total"`
		PageNum  int   `json:"pageNum"`
		PageSize int   `json:"pageSize"`
		Data     []App `json:"data"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON response")
	}

	return response.Data, nil
}

func (ps *AppService) CreateApp(organizationID, projectId, alternateID, name string, isMonoRepo bool) (App, error) {
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
		return App{}, errors.Wrap(err, "Failed to marshal request payload")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return App{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return App{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return App{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return App{}, errors.Wrap(err, "Failed to read response body")
	}

	var app App
	err = json.Unmarshal(body, &app)
	if err != nil {
		return App{}, errors.Wrap(err, "Failed to parse JSON response")
	}

	return app, nil
}

func (ps *AppService) GetApp(organizationID, appID string) (App, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/", ps.baseUrl, organizationID, appID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return App{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := ps.httpClient.Do(req)
	if err != nil {
		return App{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return App{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return App{}, errors.Wrap(err, "Failed to read response body")
	}

	var app App
	err = json.Unmarshal(body, &app)
	if err != nil {
		return App{}, errors.Wrap(err, "Failed to parse JSON response")
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
