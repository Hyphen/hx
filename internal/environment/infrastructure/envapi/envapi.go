package envapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Hyphen/cli/config"
	"github.com/Hyphen/cli/internal/environment/envvars"
	"github.com/Hyphen/cli/internal/oauth"
	"github.com/pkg/errors"
)

type EnvApi struct {
	baseUrl string
}

func New() *EnvApi {
	return &EnvApi{
		baseUrl: "http://localhost:4001",
	}
}

func (e *EnvApi) getAuthToken() (string, error) {
	creadentials, err := config.LoadCredentials()
	if err != nil {
		return "", err
	}
	if oauth.IsTokenExpired(creadentials.Default.ExpiryTime) {
		tokenResponse, err := oauth.RefreshToken(creadentials.Default.HyphenRefreshToken)
		if err != nil {
			return "", err
		}
		config.SaveCredentials(tokenResponse.AccessToken, tokenResponse.RefreshToken, tokenResponse.IDToken, tokenResponse.ExpiryTime)
		return tokenResponse.AccessToken, nil
	}
	return creadentials.Default.HyphenAccessToken, nil
}

func (e *EnvApi) Initialize(apiName, apiId string) error {
	fmt.Println("Initializing API")
	token, err := e.getAuthToken()
	if err != nil {
		// Providing user-friendly message
		return WrapError(errors.Wrap(err, "Failed to login"), "Unable to login. Please check your credentials and try again.")
	}

	url := e.baseUrl + "/apps/"
	body := map[string]string{
		"name":  apiName,
		"appId": apiId,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return WrapError(errors.Wrap(err, "Failed to marshal request body"), "Failed to prepare the request. Please try again.")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return WrapError(errors.Wrap(err, "Failed to create new request"), "Failed to create the API request. Please try again.")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return WrapError(errors.Wrap(err, "API request failed"), "Failed to communicate with the server. Please check your network connection and try again.")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		return nil
	}
	fmt.Println(resp)

	switch resp.StatusCode {
	case 401:
		return WrapError(errors.New("Unauthorized"), "Unauthorized access. Please check your API token.")
	case 409:
		return WrapError(errors.New("Conflict"), "Conflict detected. The resource may already exist.")
	default:
		fmt.Println(resp)
		return WrapError(errors.Errorf("Unexpected status code: %d", resp.StatusCode), "An unexpected error occurred. Please try again later.")
	}
}

func (e *EnvApi) UploadEnvVariable(env, appID string, envData envvars.EnviromentVarsData) error {
	token, err := e.getAuthToken()
	if err != nil {
		return WrapError(errors.Wrap(err, "Failed to login"), "Unable to login. Please check your credentials and try again.")
	}

	url := e.baseUrl + "/apps/" + appID + "/" + env
	jsonBody, err := json.Marshal(envData)
	if err != nil {
		return WrapError(errors.Wrap(err, "Failed to marshal request body"), "Failed to prepare the request. Please try again.")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return WrapError(errors.Wrap(err, "Failed to create request"), "Failed to create the API request. Please try again.")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return WrapError(errors.Wrap(err, "Failed to execute request"), "Failed to communicate with the server. Please check your network connection and try again.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		switch resp.StatusCode {
		case 401:
			return WrapError(errors.New("Unauthorized"), "Unauthorized access. Please check your API token.")
		case 409:
			return WrapError(errors.New("Conflict"), "Conflict detected. The resource may already exist.")
		case 404:
			return WrapError(errors.New("Not Found"), "The requested resource could not be found.")
		default:
			return WrapError(errors.Errorf("Unexpected status code: %d", resp.StatusCode), "An unexpected error occurred. Please try again later.")
		}
	}

	return nil
}

func (e *EnvApi) GetEncryptedVariables(env, appID string) (string, error) {

	token, err := e.getAuthToken()
	if err != nil {
		return "", WrapError(errors.Wrap(err, "Failed to login"), "Unable to login. Please check your credentials and try again.")
	}

	url := e.baseUrl + "/apps/" + appID + "/" + env
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", WrapError(errors.Wrap(err, "Failed to create request"), "Failed to create the API request. Please try again.")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", WrapError(errors.Wrap(err, "Failed to execute request"), "Failed to communicate with the server. Please check your network connection and try again.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case 401:
			return "", WrapError(errors.New("Unauthorized"), "Unauthorized access. Please check your API token.")
		case 404:
			return "", WrapError(errors.New("Not Found"), "The requested resource could not be found.")
		default:
			return "", WrapError(errors.Errorf("Unexpected status code: %d", resp.StatusCode), "An unexpected error occurred. Please try again later.")
		}
	}

	var envData envvars.EnviromentVarsData
	if err := json.NewDecoder(resp.Body).Decode(&envData); err != nil {
		return "", WrapError(errors.Wrap(err, "Failed to decode response body"), "Failed to process the server response. Please try again.")
	}

	return envData.Data, nil
}
