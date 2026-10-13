package build

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

type siteRoundTripper func(*http.Request) (*http.Response, error)

func (f siteRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCloudflareStaticBuildFlow(t *testing.T) {
	for _, scenario := range []string{"uploaded", "wrong-app-name", "wrong-revision", "wrong-uri", "manifest-revision", "missing-checksum", "renew", "renew-revision", "manifest-failed", "scope-mismatch", "redirect", "storage-mismatch", "object-mismatch", "many-files", "all-registries", "second-upload-failed"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("hello"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "empty.txt"), nil, 0600))
			for _, name := range []string{"manifest.json", "assets/e\u0301 + logo.svg", "assets/nested/app.js"} {
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("hello"), 0600))
			}
			if scenario == "many-files" {
				for i := 0; i < 26; i++ {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "asset-"+strconv.Itoa(i)+".txt"), []byte("hello"), 0600))
				}
			}
			originalTransport := http.DefaultTransport
			originalFS := config.FS
			config.FS = buildConfigFS{originalFS}
			defer func() { http.DefaultTransport = originalTransport; config.FS = originalFS }()
			var events []string
			var manifest []uploadFile
			multi := scenario == "all-registries" || scenario == "second-upload-failed"
			registryID := "conn_registry"
			var authorized []string
			completed := 0
			revision := strings.Repeat("a", 24)
			expiresAt := time.Now().Add(time.Minute)
			capability := func() map[string]any {
				uploads := []r2Upload{}
				entries := []map[string]any{}
				prefix := "revisions/website/revision_" + revision + "/"
				for _, file := range manifest {
					filePath := norm.NFC.String(file.Path)
					uploads = append(uploads, r2Upload{Path: filePath, Key: prefix + "files/" + filePath, ContentType: "text/plain"})
					entries = append(entries, map[string]any{"path": filePath, "size": file.Size, "sha256": file.SHA256, "contentType": "text/plain"})
				}
				content, err := json.Marshal(map[string]any{"formatVersion": 4, "revision": revision, "sha256": strings.Repeat("d", 64), "appName": "website", "appId": "app_test", "registryId": registryID, "files": entries})
				require.NoError(t, err)
				return map[string]any{"provider": "cloudflare", "appId": "app_test", "revision": revision, "uri": "r2://nf-sites-test/" + strings.TrimSuffix(prefix, "/"), "expiresAt": expiresAt,
					"endpoint": "https://" + strings.Repeat("a", 32) + ".r2.cloudflarestorage.com", "bucketName": "nf-sites-test",
					"credentials": r2Credentials{AccessKeyID: "test-access-key", SecretAccessKey: "temporary-test-secret", SessionToken: "test-session-token", ExpiresAt: expiresAt},
					"uploads":     uploads, "manifest": map[string]any{"key": prefix + "manifest.json", "content": string(content)},
					"limits": map[string]int{"maxFiles": 100, "maxBytes": 10000, "maxFileBytes": 1000}}
			}
			renewed := false
			http.DefaultTransport = siteRoundTripper(func(r *http.Request) (*http.Response, error) {
				events = append(events, r.Method+" "+r.URL.Path)
				response := func(status int, data any) (*http.Response, error) {
					bytes, err := json.Marshal(data)
					require.NoError(t, err)
					return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(bytes))), Request: r}, nil
				}
				if r.URL.Hostname() == "hyphen.test" {
					assert.Empty(t, r.Header.Get("X-API-Key"))
					assert.Equal(t, "Bearer hyphen-capability", r.Header.Get("Authorization"))
					switch r.URL.Path {
					case "/files":
						if scenario == "second-upload-failed" {
							return response(503, nil)
						}
						reader, err := r.MultipartReader()
						require.NoError(t, err)
						part, err := reader.NextPart()
						require.NoError(t, err)
						gz, err := gzip.NewReader(part)
						require.NoError(t, err)
						contents, err := io.ReadAll(gz)
						require.NoError(t, err)
						require.NoError(t, gz.Close())
						assert.Contains(t, []string{"", "hello"}, string(contents))
						return response(200, nil)
					case "/complete":
						completed++
						return response(200, map[string]string{"state": "sealed", "organizationId": "org_test", "projectId": "proj_test", "appId": "app_test", "revision": "abcd1234"})
					case "/abort":
						assert.Equal(t, http.MethodDelete, r.Method)
						return response(200, nil)
					default:
						t.Fatalf("unexpected Hyphen Cloud request %s", r.URL.Path)
					}
				}
				if r.URL.Hostname() != "api.test" {
					require.True(t, r2UploadHost.MatchString(r.URL.Hostname()), "unexpected external request")
					assert.Empty(t, r.Header.Get("X-API-Key"))
					assert.Contains(t, r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=test-access-key/")
					assert.Equal(t, "test-session-token", r.Header.Get("X-Amz-Security-Token"))
					assert.Empty(t, r.URL.RawQuery)
					assert.Empty(t, r.Header.Get("Content-Encoding"))
					if (scenario == "renew" || scenario == "renew-revision" || scenario == "storage-mismatch") && !renewed {
						return response(403, nil)
					}
					if scenario == "redirect" {
						resp, _ := response(307, nil)
						resp.Header.Set("Location", "https://evil.test/steal")
						return resp, nil
					}
					bytes, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					prefix := "/nf-sites-test/revisions/website/revision_" + revision + "/"
					if r.URL.Path == prefix+"manifest.json" {
						if scenario == "manifest-failed" {
							return response(500, nil)
						}
						var saved map[string]any
						require.NoError(t, json.Unmarshal(bytes, &saved))
						assert.Equal(t, "app_test", saved["appId"])
						assert.Equal(t, revision, saved["revision"])
						assert.Equal(t, strings.Repeat("d", 64), saved["sha256"])
						completed++
						return response(200, nil)
					}
					var file uploadFile
					for _, entry := range manifest {
						if r.URL.Path == prefix+"files/"+norm.NFC.String(entry.Path) {
							file = entry
							break
						}
					}
					require.NotEmpty(t, file.Path, "unexpected R2 object key %s", r.URL.Path)
					if norm.NFC.String(file.Path) == "assets/é + logo.svg" {
						assert.Equal(t, prefix+"files/assets/%C3%A9%20%2B%20logo.svg", r.URL.EscapedPath())
					}
					assert.Equal(t, file.SHA256, r.Header.Get("X-Amz-Content-Sha256"))
					if file.Path != "empty.txt" {
						assert.Equal(t, "hello", string(bytes))
						assert.Equal(t, int64(5), r.ContentLength)
					} else {
						assert.Empty(t, bytes)
						assert.Equal(t, http.NoBody, r.Body)
						assert.Zero(t, r.ContentLength)
					}
					return response(200, nil)
				}
				assert.Equal(t, "api-key-only-for-apix", r.Header.Get("X-API-Key"))
				switch {
				case r.URL.Path == "/api/organizations/org_test/integrations/connections":
					assert.Empty(t, r.URL.Query().Get("integrationTypes"))
					data := strings.ReplaceAll(readyRegistry, "hyphenCloud", "cloudflare")
					if multi {
						data += "," + strings.ReplaceAll(readyRegistry, "conn_registry", "conn_hyphen")
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":[` + data + `]}`)), Request: r}, nil
				case strings.HasSuffix(r.URL.Path, "/builds/uploads"):
					var payload struct {
						Files      []uploadFile `json:"files"`
						RegistryID string       `json:"registryId"`
						Revision   string       `json:"revision"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					manifest = payload.Files
					registryID = payload.RegistryID
					authorized = append(authorized, registryID)
					if registryID == "conn_hyphen" {
						return response(200, map[string]any{"sessionId": "hyphen-session", "revision": "abcd1234", "uri": "https://hyphen.test/v1/organizations/org_test/apps/app_test/hosting/revisions/abcd1234", "token": "hyphen-capability", "expiresAt": expiresAt, "uploadUrl": "https://hyphen.test/files", "finalizeUrl": "https://hyphen.test/complete", "abortUrl": "https://hyphen.test/abort", "limits": map[string]int{"maxFiles": 100, "maxBytes": 10000, "maxFileBytes": 1000}})
					}
					require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("changed"), 0600))
					if len(authorized) > 1 {
						renewed = true
						assert.Equal(t, revision, payload.Revision)
					} else {
						assert.Empty(t, payload.Revision)
					}
					result := capability()
					if scenario == "renew-revision" && renewed {
						revision = strings.Repeat("b", 24)
						result = capability()
					}
					if scenario == "manifest-revision" || scenario == "missing-checksum" {
						body := result["manifest"].(map[string]any)
						var content map[string]any
						require.NoError(t, json.Unmarshal([]byte(body["content"].(string)), &content))
						if scenario == "manifest-revision" {
							content["revision"] = strings.Repeat("b", 24)
						} else {
							delete(content, "sha256")
						}
						encoded, err := json.Marshal(content)
						require.NoError(t, err)
						body["content"] = string(encoded)
					}
					if scenario == "storage-mismatch" && renewed {
						result["bucketName"] = "another-bucket"
					}
					if scenario == "object-mismatch" || scenario == "wrong-app-name" {
						result["uploads"].([]r2Upload)[0].Key = "revisions/other/revision_" + revision + "/files/index.html"
					}
					if scenario == "wrong-uri" {
						result["uri"] = "r2://other-bucket/revisions/website/revision_" + revision
					}
					if scenario == "wrong-revision" {
						result["uploads"].([]r2Upload)[0].Key = "revisions/website/revision_" + strings.Repeat("b", 24) + "/files/index.html"
					}
					if scenario == "scope-mismatch" {
						result["appId"] = "another-app"
					}
					return response(200, result)
				case strings.HasSuffix(r.URL.Path, "/builds"):
					if multi {
						assert.Equal(t, "POST /complete", events[len(events)-2])
					} else {
						assert.Equal(t, "PUT /nf-sites-test/revisions/website/revision_"+revision+"/manifest.json", events[len(events)-2])
					}
					var payload models.NewBuild
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					if multi {
						require.Len(t, payload.Artifacts, 2)
						assert.Equal(t, 2, completed)
						assert.Equal(t, []string{"conn_registry", "conn_hyphen"}, authorized)
						assert.Equal(t, "hyphenCloud", payload.Artifacts[1].Target)
						assert.Equal(t, "conn_hyphen", payload.Artifacts[1].Site.RegistryID)
					} else {
						require.Len(t, payload.Artifacts, 1)
					}
					assert.Equal(t, "cloudflare", payload.Artifacts[0].Target)
					assert.Equal(t, "r2://nf-sites-test/revisions/website/revision_"+revision, payload.Artifacts[0].Site.URI)
					return response(201, map[string]string{"id": "abld_static"})
				default:
					t.Fatalf("unexpected API request %s", r.URL.Path)
					return nil, nil
				}
			})
			service := &BuildService{baseUrl: "https://api.test", httpClient: apiAuthClient{&http.Client{}}}
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			result, err := service.RunBuild(cmd, cprint.NewCPrinter(false), Options{Type: "static", Directory: dir})
			if scenario == "uploaded" || scenario == "renew" || scenario == "many-files" || scenario == "all-registries" {
				require.NoError(t, err)
				assert.Equal(t, "abld_static", result.Id)
			} else {
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "temporary-test-secret")
				assert.NotContains(t, err.Error(), "test-session-token")
				assert.NotContains(t, events, "POST /api/organizations/org_test/apps/app_test/builds")
				if scenario == "second-upload-failed" {
					assert.Equal(t, 1, completed)
					assert.Equal(t, []string{"conn_registry", "conn_hyphen"}, authorized)
					assert.Contains(t, events, "DELETE /abort")
				} else {
					assert.NotContains(t, strings.Join(events, " "), "/builds/uploads/conn_registry/")
				}
			}
			if scenario == "renew" {
				assert.True(t, renewed)
			}
			if scenario == "many-files" {
				assert.Equal(t, []string{"conn_registry"}, authorized)
			}

		})
	}
}

func TestR2DestinationsRejectUntrustedURLs(t *testing.T) {
	for _, target := range []string{"http://example.com/x?sig=x", "https://evil.test/x?sig=x", "https://user@" + strings.Repeat("a", 32) + ".r2.cloudflarestorage.com/x?sig=x", "https://" + strings.Repeat("a", 32) + ".r2.cloudflarestorage.com/x"} {
		_, err := putR2Object(context.Background(), nil, uploadCapability{Endpoint: target}, "", "", nil, 0, "")
		require.EqualError(t, err, "invalid R2 upload destination")
	}
}

func TestStaticBuildRegistersAllProviders(t *testing.T) {
	client := &http.Client{Transport: siteRoundTripper(func(r *http.Request) (*http.Response, error) {
		var body models.NewBuild
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Len(t, body.Artifacts, 2)
		assert.Equal(t, "hyphenCloud", body.Artifacts[0].Target)
		assert.Equal(t, "cloudflare", body.Artifacts[1].Target)
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"id":"build"}`))}, nil
	})}
	service := &BuildService{baseUrl: "https://api.test", httpClient: client}
	_, err := service.CreateBuild(CreateBuildOptions{OrganizationId: "org", AppId: "app", Sites: []*models.StaticArtifact{{Provider: "hyphenCloud", RegistryID: "hc", URI: "https://hyphen.test/v1/organizations/org/apps/app/hosting/revisions/12345678"}, {Provider: "cloudflare", RegistryID: "cf", URI: "r2://nf-sites-test/revisions/website/revision_" + strings.Repeat("a", 24)}}})
	require.NoError(t, err)
}

func TestR2ObjectKeysPreservePathsWithinRevision(t *testing.T) {
	prefix := "revisions/website/revision_" + strings.Repeat("a", 24) + "/"
	for _, suffix := range []string{"manifest.json", "files/index.html", "files/manifest.json", "files/assets/é + logo.svg", "files/" + strings.Repeat("a", 1024-len(prefix)-len("files/"))} {
		assert.True(t, validR2ObjectKey(prefix+suffix), suffix)
	}
	for _, suffix := range []string{"0", "files/", "files/../manifest.json", "files/./index.html", "files/assets//x", "files/x\\y", "files/%2e%2e/manifest.json", "files/x?y", "files/x#y", "files/x\x00y", "files/" + strings.Repeat("a", 1025-len(prefix)-len("files/"))} {
		assert.False(t, validR2ObjectKey(prefix+suffix), suffix)
	}
}
