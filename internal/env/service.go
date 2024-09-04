package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/conf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type EnvServicer interface {
	GetEnvironment(organizationId, appId, env string) (Environment, bool, error)
	PutEnv(organizationId, appId, environmentId string, env Env) error
	GetEnv(organizationId, appId, env string) (Env, error)
	ListEnvs(organizationId, appId string, size, page int) ([]Env, error)
	ListEnvironments(organizationId, appId string, size, page int) ([]Environment, error)
}

type EnvService struct {
	baseUrl      string
	oauthService oauth.OAuthServicer
	httpClient   httputil.Client
}

var _ EnvServicer = (*EnvService)(nil)

func NewService() *EnvService {
	baseUrl := conf.GetBaseApixUrl()

	return &EnvService{
		baseUrl:      baseUrl,
		oauthService: oauth.DefaultOAuthService(),
		httpClient:   &http.Client{},
	}

}

func (es *EnvService) GetEnvironment(organizationId, appId, environmentId string) (Environment, bool, error) {
	token, err := es.oauthService.GetValidToken()
	if err != nil {
		return Environment{}, false, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/projects/%s/environments/%s/", es.baseUrl, organizationId, appId, environmentId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Environment{}, false, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return Environment{}, false, errors.Wrap(err, "Failed to perform the HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Environment{}, false, nil
	} else if resp.StatusCode != http.StatusOK {
		return Environment{}, false, errors.HandleHTTPError(resp)
	}

	var environment Environment
	if err := json.NewDecoder(resp.Body).Decode(&environment); err != nil {
		return Environment{}, false, errors.Wrap(err, "Failed to decode the response body")
	}

	return environment, true, nil
}

func (es *EnvService) PutEnv(organizationId, appId, environmentId string, env Env) error {
	token, err := es.oauthService.GetValidToken()
	if err != nil {
		return errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	envName, err := GetEnvName(environmentId)
	if err != nil {
		return errors.Wrap(err, "Failed to get environment name")
	}

	url := fmt.Sprintf("%s/env/organizations/%s/apps/%s/environments/%s/env", es.baseUrl, organizationId, appId, envName)

	envJSON, err := json.Marshal(env)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal environment data to JSON")
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(envJSON))
	if err != nil {
		return errors.Wrap(err, "Failed to create a new HTTP request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed to perform the HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.HandleHTTPError(resp)
	}

	return nil
}

func (es *EnvService) GetEnv(organizationId, appId, envName string) (Env, error) {
	token, err := es.oauthService.GetValidToken()
	if err != nil {
		return Env{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}
	url := fmt.Sprintf("%s/organizations/%s/apps/%s/environments/%s/env", es.baseUrl, organizationId, appId, envName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Env{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return Env{}, errors.Wrap(err, "Failed to perform the HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Env{}, errors.HandleHTTPError(resp)
	}

	var envData Env
	if err := json.NewDecoder(resp.Body).Decode(&envData); err != nil {
		return Env{}, errors.Wrap(err, "Failed to decode the response body")
	}

	return envData, nil

}

func (es *EnvService) ListEnvs(organizationId, appId string, size, page int) ([]Env, error) {
	token, err := es.oauthService.GetValidToken()
	if err != nil {
		return []Env{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/env/organizations/%s/apps/%s/envs?pageSize=%d&pageNum=%d",
		es.baseUrl, organizationId, appId, size, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []Env{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return []Env{}, errors.Wrap(err, "Failed to perform the HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []Env{}, errors.HandleHTTPError(resp)
	}

	envsData := struct {
		Data       []Env `json:"data"`
		TotalCount int   `json:"totalCount"`
		PageNum    int   `json:"pageNum"`
		PageSize   int   `json:"pageSize"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&envsData); err != nil {
		return []Env{}, errors.Wrap(err, "Failed to decode response body")
	}

	return envsData.Data, nil
}

func (es *EnvService) ListEnvironments(organizationId, appId string, size, page int) ([]Environment, error) {
	token, err := es.oauthService.GetValidToken()
	if err != nil {
		return []Environment{}, errors.Wrap(err, "Failed to authenticate. Please check your credentials and try again.")
	}

	url := fmt.Sprintf("%s/api/organizations/%s/projects/%s/environments?pageSize=%d&pageNum=%d",
		es.baseUrl, organizationId, appId, size, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []Environment{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return []Environment{}, errors.Wrap(err, "Failed to perform the HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []Environment{}, errors.HandleHTTPError(resp)
	}

	envsData := struct {
		Data       []Environment `json:"data"`
		TotalCount int           `json:"totalCount"`
		PageNum    int           `json:"pageNum"`
		PageSize   int           `json:"pageSize"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&envsData); err != nil {
		return []Environment{}, errors.Wrap(err, "Failed to decode response body")
	}

	return envsData.Data, nil
}
