package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type EnvServicer interface {
	GetEnvironment(organizationId, projectId, environment string) (Environment, bool, error)
	PutEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64, env Env) error
	GetEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId *int64, version *int) (Env, error)
	ListEnvs(organizationId, appId string, size, page int) ([]Env, error)
	ListEnvVersions(organizationId, appId, environmentId string, size, page int) ([]Env, error)
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

func (es *EnvService) PutEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64, env Env) error {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dot-env/", es.baseUrl, organizationId, appId)

	query := url.Values{}
	query.Set("secretKeyId", strconv.FormatInt(secretKeyId, 10))
	if environmentId != "default" {
		query.Set("environmentId", environmentId)
	}

	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())

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

func (es *EnvService) GetEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId *int64, version *int) (Env, error) {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dot-env/", es.baseUrl, organizationId, appId)

	query := url.Values{}

	if secretKeyId != nil {
		query.Set("secretKeyId", strconv.FormatInt(*secretKeyId, 10))
	}

	if version != nil {
		query.Set("version", strconv.Itoa(*version))
	}

	if environmentId != "default" {
		query.Set("environmentId", environmentId)
	}

	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())

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
	baseURL := fmt.Sprintf("%s/api/organizations/%s/dot-envs", es.baseUrl, organizationId)

	query := url.Values{}
	query.Set("pageSize", strconv.Itoa(size))
	query.Set("pageNum", strconv.Itoa(page))
	if appId != "" {
		query.Set("appIds", appId)
	}

	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())

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

func (es *EnvService) ListEnvVersions(organizationId, appId, environmentId string, size, page int) ([]Env, error) {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dot-env/versions/", es.baseUrl, organizationId, appId)

	query := url.Values{}
	query.Set("pageSize", strconv.Itoa(size))
	query.Set("pageNum", strconv.Itoa(page))

	if environmentId != "default" {
		query.Set("environmentId", environmentId)
	}

	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())

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

	for i := range envsData.Data {
		fmt.Println(envsData.Data[i].Version)
		if envsData.Data[i].ProjectEnv != nil {
			fmt.Println(*envsData.Data[i].ProjectEnv)
		} else {
			fmt.Println("No ProjectEnv")
		}
	}

	return envsData.Data, nil
}

func (es *EnvService) ListEnvironments(organizationId, projectId string, size, page int) ([]Environment, error) {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/projects/%s/environments", es.baseUrl, organizationId, projectId)

	query := url.Values{}
	query.Set("pageSize", strconv.Itoa(size))
	query.Set("pageNum", strconv.Itoa(page))

	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())

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

func GetLocalEnvContents(envName string) (string, error) {
	envFile, err := GetFileName(envName)
	if err != nil {
		return "", err
	}

	e, err := New(envFile)
	if err != nil {
		return "", err
	}

	return e.Data, nil
}

func GetLocalEncryptedEnv(envName string, m manifest.Manifest) (Env, error) {
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
	e.SecretKeyId = &m.SecretKeyId

	return e, nil
}
