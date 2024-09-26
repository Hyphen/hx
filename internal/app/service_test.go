package app

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

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
	assert.NotNil(t, service.httpClient)
}

func TestGetListApps(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)

	service := &AppService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	responseBody := `{
		"total": 2,
		"pageNum": 1,
		"pageSize": 10,
		"data": [
			{"id": "app1", "alternateId": "alt1", "name": "app 1", "organization": {"id": "org1", "name": "Org 1"}},
			{"id": "app2", "alternateId": "alt2", "name": "app 2", "organization": {"id": "org1", "name": "Org 1"}}
		]
	}`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	apps, err := service.GetListApps("org1", "project1", 10, 1)

	assert.NoError(t, err)
	assert.Len(t, apps, 2)
	assert.Equal(t, "app1", apps[0].ID)
	assert.Equal(t, "app2", apps[1].ID)

	mockHTTPClient.AssertExpectations(t)
}

func TestCreateApp(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)

	service := &AppService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	responseBody := `{"id": "new_app", "alternateId": "alt_new", "name": "New app", "organization": {"id": "org1", "name": "Org 1"}}`

	mockResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	app, err := service.CreateApp("org1", "project1", "alt_new", "New app")

	assert.NoError(t, err)
	assert.Equal(t, "new_app", app.ID)
	assert.Equal(t, "alt_new", app.AlternateId)
	assert.Equal(t, "New app", app.Name)

	mockHTTPClient.AssertExpectations(t)
}

func TestGetApp(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)

	service := &AppService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	responseBody := `{"id": "app1", "alternateId": "alt1", "name": "app 1", "organization": {"id": "org1", "name": "Org 1"}}`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	app, err := service.GetApp("org1", "app1")

	assert.NoError(t, err)
	assert.Equal(t, "app1", app.ID)
	assert.Equal(t, "alt1", app.AlternateId)
	assert.Equal(t, "app 1", app.Name)

	mockHTTPClient.AssertExpectations(t)
}

func TestDeleteApp(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)

	service := &AppService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	mockResponse := &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

	err := service.DeleteApp("org1", "app1")

	assert.NoError(t, err)

	mockHTTPClient.AssertExpectations(t)
}

func TestErrorHandling(t *testing.T) {
	t.Run("HTTP Error", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)

		service := &AppService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		mockResponse := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
		}

		mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

		_, err := service.GetListApps("org1", "project1", 10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error: please try again later")
	})

	t.Run("JSON Parsing Error", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)

		service := &AppService{
			baseUrl:    "https://api.example.com",
			httpClient: mockHTTPClient,
		}

		mockResponse := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("Invalid JSON")),
		}

		mockHTTPClient.On("Do", mock.Anything).Return(mockResponse, nil)

		_, err := service.GetListApps("org1", "project1", 10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to parse JSON response")
	})
}

func TestAppService_HTTPClientError(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)

	service := &AppService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("error")),
	}, errors.New("network error"))

	_, err := service.GetListApps("org1", "project1", 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")

	_, err = service.CreateApp("org1", "project1", "alt1", "Test app")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")

	_, err = service.GetApp("org1", "app1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")

	err = service.DeleteApp("org1", "app1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}

func TestAppService_ReadBodyError(t *testing.T) {
	mockHTTPClient := new(httputil.MockHTTPClient)

	service := &AppService{
		baseUrl:    "https://api.example.com",
		httpClient: mockHTTPClient,
	}

	errorReader := &ErrorReader{Err: errors.New("read error")}
	mockResponseGet := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(errorReader),
	}

	mockResponseCreate := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(errorReader),
	}

	mockHTTPClient.On("Do", mock.Anything).Return(mockResponseGet, nil).Once()
	mockHTTPClient.On("Do", mock.Anything).Return(mockResponseCreate, nil).Once()
	mockHTTPClient.On("Do", mock.Anything).Return(mockResponseGet, nil).Once()

	_, err := service.GetListApps("org1", "project1", 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to read response body")

	_, err = service.CreateApp("org1", "project1", "alt1", "Test app")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to read response body")

	_, err = service.GetApp("org1", "app1")
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

func TestAppService_NewRequestError(t *testing.T) {
	service := &AppService{
		baseUrl: "://invalid-url",
	}

	_, err := service.GetListApps("org1", "project1", 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")

	_, err = service.CreateApp("org1", "project1", "alt1", "Test app")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")

	_, err = service.GetApp("org1", "app1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")

	err = service.DeleteApp("org1", "app1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create request")
}
