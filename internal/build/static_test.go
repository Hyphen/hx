package build

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

const readyRegistry = `{"id":"conn_registry","type":"SiteRegistry","status":"Ready","organization":{"id":"org_test"},"project":{"id":"proj_test"},"entity":{"id":"proj_test","type":"Project"},"organizationIntegration":{"id":"oint_cloud","type":"hyphenCloud"}}`

func TestStaticBuildFlow(t *testing.T) {
	for _, failure := range []string{"", "upload", "expired", "finalize", "wrong-receipt", "redirect", "register"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "assets"), 0700))
			contents := map[string][]byte{"index.html": []byte("<html>hello-world</html>"), "assets/script.js": []byte("console.log('hello')"), "assets/☀ light.css": []byte("body{}"), "empty.txt": {}}
			for name, data := range contents {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0600))
			}
			var manifest []uploadFile
			var events []string
			var uploadBase string
			var escapedRequests int
			redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { escapedRequests++; w.WriteHeader(200) }))
			defer redirect.Close()
			upload := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer capability-secret", r.Header.Get("Authorization"))
				assert.Empty(t, r.Header.Get("X-API-Key"))
				events = append(events, r.Method+" "+r.URL.Path)
				switch r.URL.Path {
				case "/files":
					if failure == "upload" {
						http.Error(w, "digest mismatch capability-secret", 422)
						return
					}
					if failure == "expired" {
						http.Error(w, "expired", 401)
						return
					}
					if failure == "redirect" {
						http.Redirect(w, r, redirect.URL, 307)
						return
					}
					reader, err := r.MultipartReader()
					if !assert.NoError(t, err) {
						return
					}
					part, err := reader.NextPart()
					if !assert.NoError(t, err) {
						return
					}
					// Go's FileName strips directories; read the actual disposition.
					assert.Equal(t, "gzip", part.Header.Get("Content-Encoding"))
					gz, err := gzip.NewReader(part)
					if !assert.NoError(t, err) {
						return
					}
					data, err := io.ReadAll(gz)
					assert.NoError(t, err)
					assert.NoError(t, gz.Close())
					_, disposition, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
					assert.NoError(t, err)
					matched := false
					for _, file := range manifest {
						if disposition["filename"] == file.Path {
							matched = true
							assert.Equal(t, contents[file.Path], data)
							assert.Equal(t, int64(len(data)), file.Size)
							assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256(data)), file.SHA256)
						}
					}
					assert.True(t, matched, "uploaded filename must match manifest path")
					_, err = reader.NextPart()
					assert.ErrorIs(t, err, io.EOF)
					fmt.Fprint(w, `{"files":[]}`)
				case "/complete":
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					assert.Empty(t, body)
					if failure == "finalize" {
						http.Error(w, "incomplete", 409)
						return
					}
					appID := "app_test"
					if failure == "wrong-receipt" {
						appID = "app_wrong"
					}
					json.NewEncoder(w).Encode(map[string]string{"state": "sealed", "organizationId": "org_test", "projectId": "proj_test", "appId": appID, "revision": "abcd1234"})
				case "/abort":
					assert.Equal(t, http.MethodDelete, r.Method)
					fmt.Fprint(w, `{}`)
				default:
					t.Errorf("unexpected direct request %s", r.URL)
					w.WriteHeader(404)
				}
			}))
			defer upload.Close()
			uploadBase = upload.URL
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "api-key-only-for-apix", r.Header.Get("X-API-Key"))
				assert.Empty(t, r.Header.Get("Authorization"))
				events = append(events, r.Method+" "+r.URL.Path)
				switch r.URL.Path {
				case "/api/organizations/org_test/integrations/connections":
					assert.Equal(t, "proj_test", r.URL.Query().Get("projectIds"))
					assert.Equal(t, "SiteRegistry", r.URL.Query().Get("types"))
					assert.Equal(t, "hyphenCloud", r.URL.Query().Get("integrationTypes"))
					fmt.Fprint(w, `{"data":[`+readyRegistry+`]}`)
				case "/api/organizations/org_test/apps/app_test/builds/uploads":
					var payload struct {
						RegistryID    string       `json:"registryId"`
						Files         []uploadFile `json:"files"`
						EnvironmentID string       `json:"environmentId"`
						PreviewHash   string       `json:"previewHash"`
					}
					assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					assert.Equal(t, "conn_registry", payload.RegistryID)
					assert.Equal(t, "env_test", payload.EnvironmentID)
					assert.Equal(t, "preview-hash", payload.PreviewHash)
					manifest = payload.Files
					// Later local edits must not change the already-hashed upload.
					assert.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("changed"), 0600))
					json.NewEncoder(w).Encode(map[string]any{"sessionId": "session", "revision": "abcd1234", "token": "capability-secret", "expiresAt": time.Now().Add(time.Minute), "uploadUrl": uploadBase + "/files", "finalizeUrl": uploadBase + "/complete", "abortUrl": uploadBase + "/abort", "limits": map[string]int{"maxFiles": 100, "maxBytes": 100000, "maxFileBytes": 10000}})
				case "/api/organizations/org_test/apps/app_test/builds":
					assert.Equal(t, "POST /complete", events[len(events)-2])
					assert.Equal(t, "env_test", r.URL.Query().Get("environmentId"))
					assert.Equal(t, "preview-name", r.URL.Query().Get("previewName"))
					var payload map[string]any
					assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					assert.Equal(t, []any{map[string]any{"type": "Static", "target": "hyphenCloud", "site": map[string]any{"registryId": "conn_registry", "revision": "abcd1234"}}}, payload["artifacts"])
					assert.NotEmpty(t, payload["commitSha"])
					if failure == "register" {
						http.Error(w, "registration failed", 503)
						return
					}
					w.WriteHeader(201)
					fmt.Fprint(w, `{"id":"abld_static","app":{"id":"app_test"}}`)
				default:
					t.Errorf("unexpected APIX request %s", r.URL)
					w.WriteHeader(404)
				}
			}))
			defer api.Close()
			oldFS := config.FS
			config.FS = buildConfigFS{oldFS}
			defer func() { config.FS = oldFS }()
			service := &BuildService{baseUrl: api.URL, httpClient: apiAuthClient{api.Client()}}
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			result, err := service.RunBuild(cmd, cprint.NewCPrinter(false), Options{Type: "static", Directory: dir, EnvironmentID: "env_test", Preview: "preview-name", PreviewHash: "preview-hash"})
			if failure == "" {
				require.NoError(t, err)
				assert.Equal(t, "abld_static", result.Id)
				assert.NotContains(t, events, "DELETE /abort")
			} else {
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "capability-secret")
				if failure != "register" {
					assert.NotContains(t, events, "POST /api/organizations/org_test/apps/app_test/builds")
				}
				if failure == "wrong-receipt" || failure == "register" {
					assert.NotContains(t, events, "DELETE /abort")
				} else {
					assert.Contains(t, events, "DELETE /abort")
				}
			}
			assert.Zero(t, escapedRequests)
		})
	}
}

func TestFindSiteRegistry(t *testing.T) {
	for _, test := range []struct{ name, data, want string }{
		{"ready", readyRegistry, ""},
		{"missing", "", "no ready"},
		{"pending", strings.ReplaceAll(readyRegistry, `"Ready"`, `"Pending"`), "no ready"},
		{"wrong-project", strings.ReplaceAll(readyRegistry, "proj_test", "proj_wrong"), "no ready"},
		{"wrong-org", strings.ReplaceAll(readyRegistry, "org_test", "org_wrong"), "no ready"},
		{"wrong-provider", strings.ReplaceAll(readyRegistry, "hyphenCloud", "azure"), "no ready"},
		{"ambiguous", readyRegistry + "," + strings.ReplaceAll(readyRegistry, "conn_registry", "conn_other"), "multiple ready"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"data":[`+test.data+`]}`) }))
			defer server.Close()
			service := BuildService{baseUrl: server.URL, httpClient: server.Client()}
			registry, err := service.FindSiteRegistry("org_test", "proj_test")
			if test.want != "" {
				require.ErrorContains(t, err, test.want)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "oint_cloud", registry.OrganizationIntegration.Id)
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

func TestValidateCapability(t *testing.T) {
	var valid uploadCapability
	valid.Token, valid.SessionID, valid.Revision = "token", "session", "revision"
	valid.ExpiresAt = time.Now().Add(time.Minute)
	valid.UploadURL, valid.FinalizeURL, valid.AbortURL = "https://uploads.test/files", "https://uploads.test/complete", "https://uploads.test/abort"
	valid.Limits.MaxFiles, valid.Limits.MaxBytes, valid.Limits.MaxFileBytes = 10, 100, 50
	for _, test := range []struct {
		name   string
		change func(*uploadCapability)
		files  []uploadFile
	}{
		{"expired", func(c *uploadCapability) { c.ExpiresAt = time.Now().Add(-time.Minute) }, nil},
		{"relative-url", func(c *uploadCapability) { c.UploadURL = "/files" }, nil},
		{"cross-origin", func(c *uploadCapability) { c.FinalizeURL = "https://other.test/complete" }, nil},
		{"too-many", func(c *uploadCapability) { c.Limits.MaxFiles = 0 }, []uploadFile{{Size: 1}}},
		{"file-size", func(c *uploadCapability) {}, []uploadFile{{Size: 51}}},
		{"total-size", func(c *uploadCapability) {}, []uploadFile{{Size: 50}, {Size: 50}, {Size: 1}}},
	} {
		t.Run(test.name, func(t *testing.T) { c := valid; test.change(&c); require.Error(t, c.validate(test.files)) })
	}
	require.NoError(t, valid.validate([]uploadFile{{Size: 0}, {Size: 50}}))
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

func TestCancelledUpload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := uploadCapability{Token: "secret"}
	err := c.request(ctx, &http.Client{}, http.MethodPost, "http://invalid.test/upload", nil, "", 0, nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret")
}

func TestArtifactPortsSerialization(t *testing.T) {
	data, err := json.Marshal(models.Artifact{Type: "Docker", Ports: []int{}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"Docker","ports":[]}`, string(data))
	data, err = json.Marshal(models.Artifact{Type: "Static", Target: "hyphenCloud", Site: &models.StaticArtifact{RegistryID: "conn_registry", Revision: "abcd1234"}})
	require.NoError(t, err)
	assert.NotContains(t, string(data), "ports")
	assert.NotContains(t, string(data), "image")
}
