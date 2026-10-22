package build

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiAuthClient struct{ client *http.Client }

func (c apiAuthClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-API-Key", "api-key-only-for-apix")
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.client.Do(req)
}

type buildConfigFS struct{ fsutil.FileSystem }

func (buildConfigFS) ReadFile(string) ([]byte, error) {
	return []byte(`{"organization_id":"org_test","project_id":"proj_test","app_id":"app_test","app_alternate_id":"website"}`), nil
}

const readyRegistry = `{"id":"conn_registry","type":"SiteRegistry","status":"Ready","organization":{"id":"org_test"},"project":{"id":"proj_test"},"entity":{"id":"proj_test","type":"Project"},"organizationIntegration":{"id":"oint_cloud","type":"cloudflare"}}`

func TestBuildEnvironmentFailureStopsBeforeBuilding(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
	}{
		{"missing", `{"message":"not found"}`, "resolve build environment", http.StatusNotFound},
		{"forbidden", `{"message":"forbidden"}`, "resolve build environment", http.StatusForbidden},
		{"unavailable", `{"message":"unavailable"}`, "resolve build environment", http.StatusServiceUnavailable},
		{"malformed", `not json`, "decode build environment", http.StatusOK},
		{"no-id", `{}`, "has no ID", http.StatusOK},
	} {
		for _, buildType := range []string{"static", "docker"} {
			t.Run(buildType+"/"+test.name, func(t *testing.T) {
				var requests []string
				api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests = append(requests, r.URL.Path)
					w.WriteHeader(test.status)
					fmt.Fprint(w, test.body)
				}))
				defer api.Close()
				oldFS := config.FS
				config.FS = buildConfigFS{oldFS}
				defer func() { config.FS = oldFS }()
				service := BuildService{baseUrl: api.URL, httpClient: api.Client()}
				cmd := &cobra.Command{}
				cmd.SetContext(context.Background())
				directory := ""
				if buildType == "static" {
					directory = t.TempDir()
				}
				result, err := service.RunBuild(cmd, cprint.NewCPrinter(false), Options{Type: buildType, Directory: directory, EnvironmentID: "development"})
				// Wait for the handler before reading its recorded requests.
				api.Close()
				require.ErrorContains(t, err, test.want)
				assert.Nil(t, result)
				assert.Equal(t, []string{"/api/organizations/org_test/projects/proj_test/environments/development/"}, requests)
			})
		}
	}
}

func TestFindSiteRegistries(t *testing.T) {
	for _, test := range []struct {
		name, data, want string
		ids              []string
	}{
		{"cloudflare", readyRegistry, "", []string{"conn_registry"}},
		{"retired-provider", strings.ReplaceAll(readyRegistry, "cloudflare", "hyphenCloud"), "unsupported site registry provider", nil},
		{"missing", "", "no ready", nil},
		{"pending", strings.ReplaceAll(readyRegistry, `"Ready"`, `"Pending"`), "no ready", nil},
		{"wrong-project", strings.ReplaceAll(readyRegistry, "proj_test", "proj_wrong"), "no ready", nil},
		{"wrong-org", strings.ReplaceAll(readyRegistry, "org_test", "org_wrong"), "no ready", nil},
		{"unsupported-provider", strings.ReplaceAll(readyRegistry, "cloudflare", "azure"), "unsupported site registry provider", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, r.URL.Query().Get("integrationTypes"))
				assert.Equal(t, "Ready", r.URL.Query().Get("statuses"))
				assert.Equal(t, "SiteRegistry", r.URL.Query().Get("types"))
				assert.Equal(t, "proj_test", r.URL.Query().Get("projectIds"))
				fmt.Fprint(w, `{"data":[`+test.data+`]}`)
			}))
			defer server.Close()
			service := BuildService{baseUrl: server.URL, httpClient: server.Client()}
			registries, err := service.FindSiteRegistries("org_test", "proj_test")
			if test.want != "" {
				require.ErrorContains(t, err, test.want)
			} else {
				require.NoError(t, err)
				var ids []string
				for _, registry := range registries {
					ids = append(ids, registry.ID)
				}
				assert.Equal(t, test.ids, ids)
			}
		})
	}
}

func TestPrepareSiteRejectsInvalidInput(t *testing.T) {
	for _, name := range []string{"empty", "symlink", "bad%path", "bad?path", "bad#path", "bad\\path"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if name == "symlink" {
				require.NoError(t, os.Symlink("missing", filepath.Join(dir, name)))
			} else if name != "empty" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("hello"), 0600))
			}
			_, err := prepareSite(dir, t.TempDir())
			require.Error(t, err)
		})
	}
}

func TestDockerBuildArtifactUnchanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload models.NewBuild
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Len(t, payload.Artifacts, 1)
		assert.Equal(t, "Docker", payload.Artifacts[0].Type)
		require.NotNil(t, payload.Artifacts[0].Image)
		assert.Equal(t, "registry/image:tag", payload.Artifacts[0].Image.URI)
		assert.Equal(t, []int{8080}, payload.Artifacts[0].Ports)
		assert.Nil(t, payload.Artifacts[0].Site)
		assert.Empty(t, payload.Artifacts[0].Target)
		w.WriteHeader(201)
		fmt.Fprint(w, `{"id":"abld_docker"}`)
	}))
	defer server.Close()
	service := BuildService{baseUrl: server.URL, httpClient: server.Client()}
	_, err := service.CreateBuild(CreateBuildOptions{OrganizationId: "org", AppId: "app", DockerUris: []string{"registry/image:tag"}, Ports: []int{8080}})
	require.NoError(t, err)
}

func TestArtifactPortsSerialization(t *testing.T) {
	data, err := json.Marshal(models.Artifact{Type: "Docker", Ports: []int{}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"Docker","ports":[]}`, string(data))
	data, err = json.Marshal(models.Artifact{Type: "Static", Target: "hyphenCloud", Site: &models.StaticArtifact{RegistryID: "conn_registry", URI: "https://hyphencloud.test/v1/organizations/org_test/apps/app_test/hosting/revisions/abcd1234"}})
	require.NoError(t, err)
	assert.NotContains(t, string(data), "ports")
	assert.NotContains(t, string(data), "image")
	assert.NotContains(t, string(data), `"revision"`)
	assert.Contains(t, string(data), `"uri"`)
}
