package deployment

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hyphen/cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEnvironmentDeploymentKinds(t *testing.T) {
	for _, target := range []string{"", "oint_cloud"} {
		t.Run(target, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				var payload map[string]json.RawMessage
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				assert.JSONEq(t, `{"id":"proj_test"}`, string(payload["project"]))
				assert.JSONEq(t, `{"id":"env_test"}`, string(payload["projectEnvironment"]))
				if target == "" {
					assert.Contains(t, payload, "apps")
					assert.JSONEq(t, `[{"app":{"id":"app_web"}}]`, string(payload["apps"]))
					assert.NotContains(t, payload, "sites")
				} else {
					assert.JSONEq(t, `[]`, string(payload["apps"]))
					assert.JSONEq(t, `[{"app":{"id":"app_web"},"deploymentSettings":{"targets":[{"id":"oint_cloud"}]}}]`, string(payload["sites"]))
				}
				w.WriteHeader(201)
				fmt.Fprint(w, `{"id":"dply_test","isReady":true}`)
			}))
			defer server.Close()
			service := DeploymentService{baseUrl: server.URL, httpClient: server.Client()}
			result, err := service.CreateEnvironmentDeployment("org_test", "proj_test", "env_test", "app_web", "website", "website", "", target)
			require.NoError(t, err)
			assert.Equal(t, "dply_test", result.ID)
		})
	}
}

func TestAddSitesPreservesAppsAndExistingSiteTargets(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method)
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"id":"dply_test","isReady":true,"apps":[{"app":{"id":"app_api"}}],"sites":[{"app":{"id":"app_old","name":"Old","alternateId":"old"},"deploymentSettings":{"targets":[{"id":"oint_existing","type":"hyphenCloud"}]}}]}`)
			return
		}
		var payload map[string]json.RawMessage
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Len(t, payload, 1)
		assert.NotContains(t, payload, "apps")
		assert.JSONEq(t, `[{"app":{"id":"app_old"},"deploymentSettings":{"targets":[{"id":"oint_existing"}]}},{"app":{"id":"app_new"},"deploymentSettings":{"targets":[{"id":"oint_cloud"}]}}]`, string(payload["sites"]))
		fmt.Fprint(w, `{"id":"dply_test"}`)
	}))
	defer server.Close()
	service := DeploymentService{baseUrl: server.URL, httpClient: server.Client()}
	result, err := service.AddSitesToDeployment("org_test", "dply_test", []string{"app_new"}, "oint_cloud")
	require.NoError(t, err)
	assert.True(t, result.IsReady)
	assert.Equal(t, []string{"GET", "PATCH", "GET"}, calls)
}

func TestAddAppsDoesNotReplaceSites(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"id":"dply_test","isReady":true,"apps":[{"app":{"id":"app_api"},"deploymentSettings":{"targets":[{"id":"oint_azure"}],"availability":"private","scale":"small","trafficRegions":["us"],"dns":{},"advanced":null}}],"sites":[{"app":{"id":"app_web"}}]}`)
			return
		}
		var payload map[string]json.RawMessage
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.NotContains(t, payload, "sites")
		var apps []patchDeploymentApp
		assert.NoError(t, json.Unmarshal(payload["apps"], &apps))
		if assert.Len(t, apps, 2) {
			assert.Equal(t, "oint_azure", apps[0].DeploymentSettings.Targets[0].ID)
			assert.Equal(t, "private", apps[0].DeploymentSettings.Availability)
			assert.Nil(t, apps[0].DeploymentSettings.DNS)
			assert.Equal(t, "app_new", apps[1].App.ID)
		}
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	service := DeploymentService{baseUrl: server.URL, httpClient: server.Client()}
	_, err := service.AddAppsToDeployment("org_test", "dply_test", []string{"app_new"})
	require.NoError(t, err)
}

func TestCreateRunIncludesBothKinds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Artifacts []AppSources `json:"artifacts"`
			PreviewID string       `json:"previewId"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&input))
		assert.Equal(t, "preview_test", input.PreviewID)
		assert.Equal(t, []AppSources{{AppId: "app_api", Build: "latest"}, {AppId: "app_web", BuildId: "abld_static"}}, input.Artifacts)
		json.NewEncoder(w).Encode(models.DeploymentRun{ID: "depr_test"})
	}))
	defer server.Close()
	service := DeploymentService{baseUrl: server.URL, httpClient: server.Client()}
	_, err := service.CreateRun("org_test", "dply_test", []AppSources{{AppId: "app_api", Build: "latest"}, {AppId: "app_web", BuildId: "abld_static"}}, "preview_test")
	require.NoError(t, err)
}
