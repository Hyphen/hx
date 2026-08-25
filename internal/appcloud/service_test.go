package appcloud

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewService(t *testing.T) {
	t.Run("targets_the_dev_management_api_when_HYPHEN_DEV_is_set", func(t *testing.T) {
		t.Setenv("HYPHEN_DEV", "true")

		service := NewService()

		assert.Equal(t, "https://api.dev-app.hyphen.cloud", service.baseUrl)
		assert.NotNil(t, service.httpClient)
	})

	t.Run("targets_the_prod_management_api_by_default", func(t *testing.T) {
		t.Setenv("HYPHEN_DEV", "")

		service := NewService()

		assert.Equal(t, "https://api.app.hyphen.cloud", service.baseUrl)
	})
}

func TestListApps(t *testing.T) {
	t.Run("returns_the_apps_from_a_single_page", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &AppCloudService{baseUrl: "https://api.example.com", httpClient: mockHTTPClient}
		responseBody := `{
			"items": [
				{"id": "the_app_id", "owner": "drew", "name": "the app", "domains": ["the.app.hyphen.cloud"]},
				{"id": "another_app", "owner": "drew", "name": "another", "domains": []}
			],
			"nextCursor": null
		}`
		mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(responseBody)),
		}, nil).Once()

		apps, err := service.ListApps("the_org_id", "the_project_id")

		assert.NoError(t, err)
		assert.Len(t, apps, 2)
		assert.Equal(t, "the_app_id", apps[0].ID)
		assert.Equal(t, []string{"the.app.hyphen.cloud"}, apps[0].Domains)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("follows_the_cursor_across_pages", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &AppCloudService{baseUrl: "https://api.example.com", httpClient: mockHTTPClient}
		firstPage := `{"items": [{"id": "app_1"}], "nextCursor": "cursor_2"}`
		secondPage := `{"items": [{"id": "app_2"}], "nextCursor": null}`
		mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(firstPage)),
		}, nil).Once()
		mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(secondPage)),
		}, nil).Once()

		apps, err := service.ListApps("the_org_id", "")

		assert.NoError(t, err)
		assert.Len(t, apps, 2)
		assert.Equal(t, "app_1", apps[0].ID)
		assert.Equal(t, "app_2", apps[1].ID)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("returns_an_error_when_the_api_responds_with_a_non_200", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &AppCloudService{baseUrl: "https://api.example.com", httpClient: mockHTTPClient}
		mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"error":"forbidden"}`)),
		}, nil).Once()

		apps, err := service.ListApps("the_org_id", "")

		assert.Error(t, err)
		assert.Nil(t, apps)
		mockHTTPClient.AssertExpectations(t)
	})
}

func TestGetApp(t *testing.T) {
	t.Run("returns_the_requested_app", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &AppCloudService{baseUrl: "https://api.example.com", httpClient: mockHTTPClient}
		responseBody := `{"id": "the_app_id", "owner": "drew", "name": "the app", "domains": ["the.app.hyphen.cloud"], "activeRevision": "abc123"}`
		mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(responseBody)),
		}, nil).Once()

		app, err := service.GetApp("the_app_id")

		assert.NoError(t, err)
		assert.Equal(t, "the_app_id", app.ID)
		assert.Equal(t, "abc123", app.ActiveRevision)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("returns_an_error_when_the_app_is_not_found", func(t *testing.T) {
		mockHTTPClient := new(httputil.MockHTTPClient)
		service := &AppCloudService{baseUrl: "https://api.example.com", httpClient: mockHTTPClient}
		mockHTTPClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{"error":"app not found"}`)),
		}, nil).Once()

		_, err := service.GetApp("missing_app")

		assert.Error(t, err)
		mockHTTPClient.AssertExpectations(t)
	})
}
