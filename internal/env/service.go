package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

type EnvServicer interface {
	GetEnvironment(organizationId, projectId, environment string) (models.Environment, bool, error)
	PutEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64, env models.Env) error
	GetEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId *int64, version *int) (models.Env, error)
	ListEnvs(organizationId, appId string, size, page int) ([]models.Env, error)
	ListEnvVersions(organizationId, appId, environmentId string, size, page int) ([]models.Env, error)
	ListEnvironments(organizationId, projectId string, size, page int) ([]models.Environment, error)
}

type EnvService struct {
	baseApixUrl    string
	baseHorizonUrl string
	httpClient     httputil.Client
}

var _ EnvServicer = (*EnvService)(nil)

func NewService() *EnvService {

	return &EnvService{
		baseApixUrl:    apiconf.GetBaseApixUrl(),
		baseHorizonUrl: apiconf.GetBaseHorizonUrl(),
		httpClient:     httputil.NewHyphenHTTPClient(),
	}
}

func (es *EnvService) GetEnvironment(organizationId, projectId, environmentId string) (models.Environment, bool, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/projects/%s/environments/%s/", es.baseHorizonUrl, organizationId, projectId, environmentId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return models.Environment{}, false, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return models.Environment{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return models.Environment{}, false, nil
	} else if resp.StatusCode != http.StatusOK {
		return models.Environment{}, false, errors.HandleHTTPError(resp)
	}

	var environment models.Environment
	if err := json.NewDecoder(resp.Body).Decode(&environment); err != nil {
		return models.Environment{}, false, errors.Wrap(err, "Failed to decode the response body")
	}

	return environment, true, nil
}

func (es *EnvService) PutEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64, env models.Env) error {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dot-env/", es.baseApixUrl, organizationId, appId)

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

func (es *EnvService) GetEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId *int64, version *int) (models.Env, error) {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dot-env/", es.baseHorizonUrl, organizationId, appId)

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
		return models.Env{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return models.Env{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Env{}, errors.HandleHTTPError(resp)
	}

	var envData models.Env
	if err := json.NewDecoder(resp.Body).Decode(&envData); err != nil {
		return models.Env{}, errors.Wrap(err, "Failed to decode the response body")
	}

	return envData, nil
}

func (es *EnvService) ListEnvs(organizationId, appId string, size, page int) ([]models.Env, error) {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/dot-envs", es.baseHorizonUrl, organizationId)

	query := url.Values{}
	query.Set("pageSize", strconv.Itoa(size))
	query.Set("pageNum", strconv.Itoa(page))
	if appId != "" {
		query.Set("appIds", appId)
	}

	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []models.Env{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return []models.Env{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []models.Env{}, errors.HandleHTTPError(resp)
	}

	var envsData models.PaginatedResponse[models.Env]

	if err := json.NewDecoder(resp.Body).Decode(&envsData); err != nil {
		return []models.Env{}, errors.Wrap(err, "Failed to decode response body")
	}

	return envsData.Data, nil
}

func (es *EnvService) ListEnvVersions(organizationId, appId, environmentId string, size, page int) ([]models.Env, error) {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dot-env/versions/", es.baseHorizonUrl, organizationId, appId)

	query := url.Values{}
	query.Set("pageSize", strconv.Itoa(size))
	query.Set("pageNum", strconv.Itoa(page))

	if environmentId != "default" {
		query.Set("environmentId", environmentId)
	}

	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []models.Env{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return []models.Env{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []models.Env{}, errors.HandleHTTPError(resp)
	}

	var envsData models.PaginatedResponse[models.Env]

	if err := json.NewDecoder(resp.Body).Decode(&envsData); err != nil {
		return []models.Env{}, errors.Wrap(err, "Failed to decode response body")
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

func (es *EnvService) ListEnvironments(organizationId, projectId string, size, page int) ([]models.Environment, error) {
	baseURL := fmt.Sprintf("%s/api/organizations/%s/projects/%s/environments", es.baseHorizonUrl, organizationId, projectId)

	query := url.Values{}
	query.Set("pageSize", strconv.Itoa(size))
	query.Set("pageNum", strconv.Itoa(page))

	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []models.Environment{}, errors.Wrap(err, "Failed to create a new HTTP request")
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return []models.Environment{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []models.Environment{}, errors.HandleHTTPError(resp)
	}

	var envsData models.PaginatedResponse[models.Environment]

	if err := json.NewDecoder(resp.Body).Decode(&envsData); err != nil {
		return []models.Environment{}, errors.Wrap(err, "Failed to decode response body")
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

func GetLocalEncryptedEnv(envName string, envCompletePath *string, s models.Secret) (models.Env, error) {
	envFile, err := GetFileName(envName)
	if err != nil {
		return models.Env{}, err
	}
	if envCompletePath != nil {
		envFile = path.Join(*envCompletePath, envFile)
	}

	e, err := New(envFile)
	if err != nil {
		return models.Env{}, err
	}

	envEncrytedData, err := e.EncryptData(s)
	if err != nil {
		return models.Env{}, err
	}
	e.Data = envEncrytedData
	e.SecretKeyId = &s.SecretKeyId

	return e, nil
}
