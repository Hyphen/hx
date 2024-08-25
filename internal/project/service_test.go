package project

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/oauth"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewService(t *testing.T) {
	os.Setenv("HYPHEN_CUSTOM_APIX", "https://custom-api.example.com")
	defer os.Unsetenv("HYPHEN_CUSTOM_APIX")

	service := NewService()
	assert.Equal(t, "https://custom-api.example.com", service.baseUrl)
	assert.NotNil(t, service.oauthService)
	assert.NotNil(t, service.httpClient)
}

func TestGetListProjects(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &ProjectService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	responseBody := `{
		"total": 2,
		"pageNum": 1,
		"pageSize": 10,
		"data": [
			{"id": "project1", "alternateId": "alt1", "name": "Project 1", "organization": {"id": "org1", "name": "Org 1"}},
			{"id": "project2", "alternateId": "alt2", "name": "Project 2", "organization": {"id": "org1", "name": "Org 1"}}
		]
	}`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	projects, err := service.GetListProjects("org1", 10, 1)

	assert.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "project1", projects[0].ID)
	assert.Equal(t, "project2", projects[1].ID)

	mockOAuthService.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestCreateProject(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &ProjectService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	responseBody := `{"id": "new_project", "alternateId": "alt_new", "name": "New Project", "organization": {"id": "org1", "name": "Org 1"}}`

	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	project, err := service.CreateProject("org1", "alt_new", "New Project")

	assert.NoError(t, err)
	assert.Equal(t, "new_project", project.ID)
	assert.Equal(t, "alt_new", project.AlternateId)
	assert.Equal(t, "New Project", project.Name)

	mockOAuthService.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestGetProject(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &ProjectService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	responseBody := `{"id": "project1", "alternateId": "alt1", "name": "Project 1", "organization": {"id": "org1", "name": "Org 1"}}`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	project, err := service.GetProject("org1", "project1")

	assert.NoError(t, err)
	assert.Equal(t, "project1", project.ID)
	assert.Equal(t, "alt1", project.AlternateId)
	assert.Equal(t, "Project 1", project.Name)

	mockOAuthService.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestDeleteProject(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &ProjectService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	mockResponse := &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	err := service.DeleteProject("org1", "project1")

	assert.NoError(t, err)

	mockOAuthService.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestErrorHandling(t *testing.T) {
	t.Run("Authentication Error", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		mockOAuthService := new(oauth.MockOAuthService)

		service := &ProjectService{
			baseUrl:      "https://api.example.com",
			oauthService: mockOAuthService,
			httpClient:   mockHTTPClient,
		}

		mockOAuthService.On("GetValidToken").Return("", errors.New("auth error"))

		_, err := service.GetListProjects("org1", 10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to authenticate")
	})

	t.Run("HTTP Error", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		mockOAuthService := new(oauth.MockOAuthService)

		service := &ProjectService{
			baseUrl:      "https://api.example.com",
			oauthService: mockOAuthService,
			httpClient:   mockHTTPClient,
		}

		mockOAuthService.On("GetValidToken").Return("valid_token", nil)

		mockResponse := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
		}

		mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

		_, err := service.GetListProjects("org1", 10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error: please try again later")
	})

	t.Run("JSON Parsing Error", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		mockOAuthService := new(oauth.MockOAuthService)

		service := &ProjectService{
			baseUrl:      "https://api.example.com",
			oauthService: mockOAuthService,
			httpClient:   mockHTTPClient,
		}

		mockOAuthService.On("GetValidToken").Return("valid_token", nil)

		mockResponse := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("Invalid JSON")),
		}

		mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

		_, err := service.GetListProjects("org1", 10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to parse JSON response")
	})
}

func TestProjectService_HTTPClientError(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &ProjectService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)
	mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("error")),
	}, errors.New("network error"))

	_, err := service.GetListProjects("org1", 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to send request")

	_, err = service.CreateProject("org1", "alt1", "Test Project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to send request")

	_, err = service.GetProject("org1", "project1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to send request")

	err = service.DeleteProject("org1", "project1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to send request")
}

func TestProjectService_ReadBodyError(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)
	mockOAuthService := new(oauth.MockOAuthService)

	service := &ProjectService{
		baseUrl:      "https://api.example.com",
		oauthService: mockOAuthService,
		httpClient:   mockHTTPClient,
	}

	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	errorReader := &ErrorReader{Err: errors.New("read error")}
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(errorReader),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	_, err := service.GetListProjects("org1", 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to read response body")

	_, err = service.CreateProject("org1", "alt1", "Test Project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to read response body")

	_, err = service.GetProject("org1", "project1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to read response body")
}

// ErrorReader is a custom io.Reader that always returns an error
type ErrorReader struct {
	Err error
}

func (er *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, er.Err
}

func TestProjectService_NewRequestError(t *testing.T) {
	service := &ProjectService{
		baseUrl: "://invalid-url",
	}

	mockOAuthService := new(oauth.MockOAuthService)
	service.oauthService = mockOAuthService
	mockOAuthService.On("GetValidToken").Return("valid_token", nil)

	_, err := service.GetListProjects("org1", 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")

	_, err = service.CreateProject("org1", "alt1", "Test Project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")

	_, err = service.GetProject("org1", "project1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")

	err = service.DeleteProject("org1", "project1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")
}
