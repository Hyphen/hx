package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type EnvServicer interface {
	GetEnvironment(organizationId, projectId, environment string) (Environment, bool, error)
	PutEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64, env Env) error
	GetEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64) (Env, error)
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

// PutEnvironmentEnv updates the environment variables for the given organization, app, and environment.
// In case of key rotation set the new key in the env.secretKeyId and send the current secretKeyId in the func param
func (es *EnvService) PutEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64, env Env) error {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dot-env/", es.baseUrl, organizationId, appId)

	var queryParams []string
	queryParams = append(queryParams, fmt.Sprintf("secretKeyId=%s", strconv.FormatInt(secretKeyId, 10)))
	if environmentId != "default" {
		queryParams = append(queryParams, fmt.Sprintf("environmentId=%s", environmentId))
	}
	if len(queryParams) > 0 {
		url = fmt.Sprintf("%s?%s", url, strings.Join(queryParams, "&"))
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

func (es *EnvService) GetEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64) (Env, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dot-env/", es.baseUrl, organizationId, appId)

	var queryParams []string
	queryParams = append(queryParams, fmt.Sprintf("secretKeyId=%s", strconv.FormatInt(secretKeyId, 10)))
	if environmentId != "default" {
		queryParams = append(queryParams, fmt.Sprintf("environmentId=%s", environmentId))
	}
	if len(queryParams) > 0 {
		url = fmt.Sprintf("%s?%s", url, strings.Join(queryParams, "&"))
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
	url := fmt.Sprintf("%s/api/organizations/%s/dot-envs", es.baseUrl, organizationId)

	var queryParams []string

	queryParams = append(queryParams, fmt.Sprintf("pageSize=%d", size))
	queryParams = append(queryParams, fmt.Sprintf("pageNum=%d", page))

	if appId != "" {
		queryParams = append(queryParams, fmt.Sprintf("appIds=%s", appId))
	}

	if len(queryParams) > 0 {
		url = fmt.Sprintf("%s?%s", url, strings.Join(queryParams, "&"))
	}

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

// GetLocalEncryptedEnv returns the local environment variables from the .env file
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
