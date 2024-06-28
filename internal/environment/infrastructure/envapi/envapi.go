package envapi

import (
	"bytes"
	"encoding/json"
	"net/http"

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

func (e *EnvApi) login() (string, error) {
	// Simulate login error
	return "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJRX1pObEt0YTF2bWk0cWN1TU5mVWlfX1lKYVZ6MEZiUGt3WVQ5anB4di1ZIn0.eyJleHAiOjE3MTk1OTU0MjQsImlhdCI6MTcxOTU5MzYyNCwiYXV0aF90aW1lIjoxNzE5NTkzNjI0LCJqdGkiOiJkMTdhOGRkZi00MWNjLTRkYmEtOTA5Zi1mNzRlODVlNDZkNDYiLCJpc3MiOiJodHRwczovL2F1dGguaHlwaGVuLmFpL3JlYWxtcy9kZXYiLCJhdWQiOiJhY2NvdW50Iiwic3ViIjoiM2QyMjBlOTYtZTI0MC00ZTI3LTlmZWYtYjdhZTM2YWYzYzY2IiwidHlwIjoiQmVhcmVyIiwiYXpwIjoiZW5naW5lIiwic2Vzc2lvbl9zdGF0ZSI6IjY2Y2RhNTNmLWNjMDAtNDc3NC1iNzFlLWEzNmNkNWUxYmI4YiIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiaHR0cHM6Ly9hcHAuaHlwaGVuLmFpIiwiaHR0cHM6Ly9kZXYtYXBpLmh5cGhlbi5haS8qIiwiaHR0cDovL2VuZ2luZS5sb2NhbGhvc3QvKiIsImh0dHBzOi8vZW5naW5lLmRldi5oeXBoZW4uYWkiLCJodHRwOi8vZW5naW5lLmxvY2FsaG9zdCIsImh0dHBzOi8vKi5oeXBoZW4tYXBwLnBhZ2VzLmRldiIsImh0dHBzOi8vZGV2LWFwaS5oeXBoZW4uYWkiLCJodHRwOi8vbG9jYWxob3N0OjMwMDAiLCJodHRwOi8vbG9jYWxob3N0OjQwMDAiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbImRlZmF1bHQtcm9sZXMtZGV2Iiwib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7ImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoib3BlbmlkIGVtYWlsIG9mZmxpbmVfYWNjZXNzIHByb2ZpbGUiLCJzaWQiOiI2NmNkYTUzZi1jYzAwLTQ3NzQtYjcxZS1hMzZjZDVlMWJiOGIiLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlJoaW5vIFRlc3QiLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJsdWlzLm1pcmFuZGFAcmhpbm9sYWJzLmNvIiwiZ2l2ZW5fbmFtZSI6IlJoaW5vIiwiZmFtaWx5X25hbWUiOiJUZXN0IiwiZW1haWwiOiJsdWlzLm1pcmFuZGFAcmhpbm9sYWJzLmNvIn0.RqvsfLQlzqvxzdAvSZHC87-K39K1rjUvljr_XSceJtfh_hw5GwsMPNbpEtkBuSeqaB9FopNrCvJO0n3qh_g3hMOQMoJnu4APQMwkHEfvMMDbueSDzTy9GOUb7pwMwL3gd6NS83RhT68tASULFICrlx8F7bXHUHXJjpBoPhlOlPiAotJNCaNr5XPcpgJOHhm1_SQrEK8iJDxluQObtFTZliyOG6TqkAJMc64mB8Qfz9zIXwMJP1nCowcrm9EpY8sCYOLo-kdoUnE0NTr7HTAjOf72r6UC8ZhBDF7siRjmK-z10oAB7f0Zj7qiz7F9sRtBhCJA_HsmJg26Vpnr9RwPrw", nil
}

func (e *EnvApi) Initialize(apiName, apiId string) error {
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

	switch resp.StatusCode {
	case 401:
		return WrapError(errors.New("Unauthorized"), "Unauthorized access. Please check your API token.")
	case 409:
		return WrapError(errors.New("Conflict"), "Conflict detected. The resource may already exist.")
	default:
		return WrapError(errors.Errorf("Unexpected status code: %d", resp.StatusCode), "An unexpected error occurred. Please try again later.")
	}
}

func (e *EnvApi) UploadEnvVariable(env, encryptedVars string) error {
	return nil
}

func (e *EnvApi) GetEncryptedVariables(env string) (string, error) {
	return "", nil

}
