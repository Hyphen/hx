package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type EnvServicer interface {
	GetEnvironment(organizationId, projectId, environment string) (Environment, bool, error)
	PutEnvironmentEnv(organizationId, appId, environmentId string, env Env) error
	GetEnvironmentEnv(organizationId, appId, env string) (Env, error)
	ListEnvs(organizationId, appId string, size, page int) ([]Env, error)
	ListEnvironments(organizationId, projectId string, size, page int) ([]Environment, error)
}

type EnvService struct {
	baseUrl    string
	httpClient httputil.Client
}

var _ EnvServicer = (*EnvService)(nil)

func NewService() *EnvService {
	baseUrl := apiconf.GetBaseApixUrl()

	return &EnvService{
		baseUrl:    baseUrl,
		httpClient: httputil.NewHyphenHTTPClient(),
	}

}

func (es *EnvService) GetEnvironment(organizationId, projectId, environmentId string) (Environment, bool, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/projects/%s/environments/%s/", es.baseUrl, organizationId, projectId, environmentId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Environment{}, false, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return Environment{}, false, err
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

func (es *EnvService) PutEnvironmentEnv(organizationId, appId, environmentId string, env Env) error {
	url := ""
	if environmentId == "default" {
		url = fmt.Sprintf("%s/api/organizations/%s/apps/%s/env", es.baseUrl, organizationId, appId)
	} else {
		envName, err := GetEnvName(environmentId)
		if err != nil {
			return errors.Wrap(err, "Failed to get environment name")
		}

		url = fmt.Sprintf("%s/api/organizations/%s/apps/%s/environments/%s/env", es.baseUrl, organizationId, appId, envName)
	}

	envJSON, err := json.Marshal(env)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal environment data to JSON")
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(envJSON))
	if err != nil {
		return errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.HandleHTTPError(resp)
	}

	return nil
}

func (es *EnvService) GetEnvironmentEnv(organizationId, appId, environmentId string) (Env, error) {
	url := ""
	if environmentId == "default" {
		url = fmt.Sprintf("%s/api/organizations/%s/apps/%s/env", es.baseUrl, organizationId, appId)
	} else {
		envName, err := GetEnvName(environmentId)
		if err != nil {
			return Env{}, errors.Wrap(err, "Failed to get environment name")
		}

		url = fmt.Sprintf("%s/api/organizations/%s/apps/%s/environments/%s/env", es.baseUrl, organizationId, appId, envName)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Env{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return Env{}, err
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
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/envs?pageSize=%d&pageNum=%d",
		es.baseUrl, organizationId, appId, size, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []Env{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return []Env{}, err
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

func (es *EnvService) ListEnvironments(organizationId, projectId string, size, page int) ([]Environment, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/projects/%s/environments?pageSize=%d&pageNum=%d",
		es.baseUrl, organizationId, projectId, size, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []Environment{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return []Environment{}, err
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

func GetLocalEnv(envName string, m manifest.Manifest) (Env, error) {
	envFile, err := GetFileName(envName)
	if err != nil {
		return Env{}, err
	}

	e, err := New(envFile)
	if err != nil {
		return Env{}, err
	}

	envEncrytedData, err := e.EncryptData(m.GetSecretKey())
	if err != nil {
		return Env{}, err
	}
	e.Data = envEncrytedData

	return e, nil
}
