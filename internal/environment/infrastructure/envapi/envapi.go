package envapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Hyphen/cli/internal/environment/envvars"
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

// TODO: this should come from the engine package
func (e *EnvApi) login() (string, error) {
	return "eyJhbGciOiJSUzI1NiIsImtpZCI6IjI3OWNmYWY4LTY5YjctNDJjYS1iYjE4LTNlNDNmYTJjYmVlYyIsInR5cCI6IkpXVCJ9.eyJhdWQiOltdLCJjbGllbnRfaWQiOiI1MDhhZTViZS04YTdmLTRkY2YtOTY4NS1lMDlmMGE2YjY4ZmUiLCJlbWFpbCI6Imx1aXMubWlyYW5kYUByaGlub2xhYnMuY28iLCJleHAiOjE3MTk5NjAwMjAsImZhbWlseV9uYW1lIjoiTWlyYW5kYSIsImdpdmVuX25hbWUiOiJMdWlzIiwiaWF0IjoxNzE5OTU2NDIwLCJpc19oeXBoZW5faW50ZXJuYWwiOmZhbHNlLCJpc3MiOiJodHRwczovL2Rldi1hdXRoLmh5cGhlbi5haSIsImp0aSI6IjJlNzE5NTk2LTZhNTMtNDljMy1hZjBkLWVjYmE2MDdkN2Q4NyIsIm5iZiI6MTcxOTk1NjQyMCwic2NwIjpbIm9wZW5pZCIsIm9mZmxpbmVfYWNjZXNzIiwicHJvZmlsZSIsImVtYWlsIl0sInN1YiI6IjI4MjI0NWYwLWRlZDYtNGMyOC05MGJhLTRlMmRlZmFiYmJkNiJ9.aYcSs7t8SMw6fv405xBjupddtkeXCGYSkK1ia04lPEufpss-pOWt82fgAQ8VaqEdagkuGOI8mZynQuKukKLYXBf4R0QAZWa4VTDdSukPl0cocdVpnfzOP6htDphPy5YaO_vCZ__DMgUz-SbHLzyLRXGYuYHx9GKBgJAIe75qE6eIdkUCtjWEY1ELHIKXgUQP8T6u_gYM1IBH0m4Hlt1t__9MLcsD1G_bL3srsq-vgddGpCcvpHXB0UEa0nHWH4r1opyJth3s33bOndQRQ09S961TDIxS-fILIu7RliG9MP2rZanyANf71zyU2tuYk0Pxsij0yvHrAPhvoe0TXRkaqq4LaFqZFwCECwIeayKwvDXnhCIfNC0Yk8dQpcXjCOuu7tqe_W45LYlczEceDvZ_3pBpx0DI_u7FexoFdRXX1xuHXzS3iCL5-QraAYPa5fDql3LHn7Wu2quKrf12qNYl6jRuOM6FsGPDYlozFkBXRKHnJwE83smqGZFkWgWYiJsXttgjeHL_YrmP7zCuZBZROOeE39_4c3mDgjfqMMh9eS4YiIRmMVv1emFsV11BQFqVjMLakhd8XjeqE1xkHd9A7LGA4YEcM6dMg0DRMiuXM4qR_IbPTqdw7WRJfP70SxmrI0Oe7Dchly5NpG_hX4Y5td1eos-d6xyo6zFlSvQxa0Y", nil
}

func (e *EnvApi) Initialize(apiName, apiId string) error {
	fmt.Println("Initializing API")
	token, err := e.login()
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
	token, err := e.login()
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

	token, err := e.login()
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
