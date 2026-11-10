package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/dockerutil"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func registryFixture() models.ContainerRegistry {
	registry := models.ContainerRegistry{Id: "proj_1", Name: "Shop", Url: "registry.cloudflare.com/" + strings.Repeat("a", 32)}
	registry.Auth.Server = "registry.cloudflare.com"
	registry.Auth.Username = "user"
	registry.Auth.Password = "credential-sentinel"
	return registry
}

func TestCloudflarePlatformDigest(t *testing.T) {
	for _, repository := range []string{"shop/api", "shop--test/api__v2", "p-" + strings.Repeat("b", 24) + "-a-" + strings.Repeat("c", 24)} {
		t.Run(repository, func(t *testing.T) {
			registry := registryFixture()
			destination := registry.Url + "/" + repository + ":build"
			manifest := `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`
			sum := sha256.Sum256([]byte(manifest))
			digest := "sha256:" + hex.EncodeToString(sum[:])
			index := fmt.Sprintf(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[{"digest":%q,"platform":{"os":"linux","architecture":"amd64"}}]}`, digest)
			client := new(httputil.MockHTTPClient)
			client.On("Do", mock.Anything).Run(func(args mock.Arguments) {
				request := args.Get(0).(*http.Request)
				username, password, ok := request.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "user", username)
				assert.Equal(t, "credential-sentinel", password)
				assert.Equal(t, "registry.cloudflare.com", request.URL.Host)
				assert.Equal(t, "/v2/"+strings.Repeat("a", 32)+"/"+repository+"/manifests/build", request.URL.Path)
			}).Return(&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(index))}, nil).Once()
			client.On("Do", mock.Anything).Return(&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(manifest))}, nil).Once()
			uri, err := resolveCloudflareDigest(context.Background(), client, destination, registry)
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSuffix(destination, ":build")+"@"+digest, uri)
			client.AssertExpectations(t)
		})
	}
}

func TestCloudflareDigestRejectsMalformedOrWrongPlatform(t *testing.T) {
	for _, body := range []string{
		"credential-sentinel",
		`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[{"digest":"sha256:bad","platform":{"os":"linux","architecture":"arm64"}}]}`,
	} {
		client := new(httputil.MockHTTPClient)
		client.On("Do", mock.Anything).Return(&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil)
		registry := registryFixture()
		_, err := resolveCloudflareDigest(context.Background(), client, registry.Url+"/shop/api:build", registry)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "credential-sentinel")
	}
}

func TestRegistryCredentialErrorsAreSanitized(t *testing.T) {
	for _, status := range []int{200, 403, 500} {
		client := new(httputil.MockHTTPClient)
		client.On("Do", mock.Anything).Return(&http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("credential-sentinel"))}, nil)
		service := &BuildService{baseUrl: "https://api.example.test", httpClient: client}
		_, err := service.FindRegistryConnections(context.Background(), "org_1", "proj_1")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "credential-sentinel")
	}
}

func TestCloudflareUploadDestinations(t *testing.T) {
	base := registryFixture().Url
	for _, tc := range []struct {
		registry, project, app, tag string
		valid                       bool
	}{
		{base, "shop", "api", "build", true},
		{base, "shop--test", "api__v2", "v1.2", true},
		{base, "shop/..", "api", "build", false},
		{base, "shop", "", "build", false},
		{base, "shop", "api/extra", "build", false},
		{base, "shop%2Fapi", "api", "build", false},
		{base, "Shop", "api", "build", false},
		{base, "shop", "api", "build?query", false},
		{base, "shop", "api", "build\n", false},
		{"registry.cloudflare.com.evil.test/" + strings.Repeat("a", 32), "shop", "api", "build", false},
	} {
		t.Run(tc.project+"/"+tc.app+":"+tc.tag, func(t *testing.T) {
			uri, err := cloudflareImageURI(tc.registry, tc.project, tc.app, tc.tag)
			if tc.valid {
				require.NoError(t, err)
				assert.Equal(t, base+"/"+tc.project+"/"+tc.app+":"+tc.tag, uri)
			} else {
				require.Error(t, err)
				assert.Empty(t, uri)
			}
		})
	}
	assert.True(t, isCloudflareRegistry(base))
	for _, other := range []string{"registry.azurecr.io", "us-docker.pkg.dev/project/repository", "123.dkr.ecr.us-east-1.amazonaws.com/repo", "registry.cloudflare.com.evil.test/account"} {
		assert.False(t, isCloudflareRegistry(other))
	}
}

func TestCloudflareBuildUsesRegistryDiscovery(t *testing.T) {
	for _, scenario := range []string{"upload", "push-expired", "manifest-expired", "registry-changed", "missing-credentials"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			script := `#!/bin/sh
printf '%s\n' "$@" >> "$DOCKER_TEST_DIR/args"
if [ "$1" = inspect ]; then printf '%s' '[{"Config":{"ExposedPorts":{"80/tcp":{}}}}]'; fi
if [ "$3" = login ]; then /bin/cat >> "$DOCKER_TEST_DIR/logins"; fi
if [ "$3" = push ] && [ "$FAIL_PUSH" = yes ] && [ ! -f "$DOCKER_TEST_DIR/failed" ]; then
  : > "$DOCKER_TEST_DIR/failed"
  echo unauthorized >&2
  exit 1
fi
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("DOCKER_TEST_DIR", dir)
			t.Setenv("FAIL_PUSH", "no")
			if scenario == "push-expired" {
				t.Setenv("FAIL_PUSH", "yes")
			}
			registry := registryFixture()
			manifest := `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`
			sum := sha256.Sum256([]byte(manifest))
			artifact := registry.Url + "/shop/api@sha256:" + hex.EncodeToString(sum[:])
			registryCalls, manifestCalls, buildCalls := 0, 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/organizations/org_1/deployments/containerRegistries":
					assert.Equal(t, "GET", r.Method)
					assert.Equal(t, "proj_1", r.URL.Query().Get("projectId"))
					registryCalls++
					refreshed := registry
					refreshed.Auth.Password = fmt.Sprintf("credential-%d", registryCalls)
					if registryCalls > 1 && scenario == "registry-changed" {
						refreshed.Url = "registry.cloudflare.com/" + strings.Repeat("b", 32)
					}
					if scenario == "missing-credentials" {
						refreshed.Auth.Password = ""
					}
					json.NewEncoder(w).Encode([]models.ContainerRegistry{refreshed})
				case "/api/organizations/org_1/apps/app_1/builds":
					buildCalls++
					var build models.NewBuild
					require.NoError(t, json.NewDecoder(r.Body).Decode(&build))
					require.Len(t, build.Artifacts, 1)
					assert.Equal(t, artifact, build.Artifacts[0].Image.URI)
					w.WriteHeader(201)
					fmt.Fprint(w, `{"id":"build_1"}`)
				default:
					t.Errorf("unexpected API request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			transport := http.DefaultTransport
			client := server.Client()
			client.Transport = transport
			http.DefaultTransport = siteRoundTripper(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "registry.cloudflare.com", r.URL.Host)
				_, password, _ := r.BasicAuth()
				assert.Equal(t, fmt.Sprintf("credential-%d", registryCalls), password)
				manifestCalls++
				status := 200
				if scenario == "manifest-expired" && manifestCalls == 1 {
					status = 401
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(manifest))}, nil
			})
			defer func() { http.DefaultTransport = transport }()
			projectID, projectSlug, appSlug := "proj_1", "shop", "api"
			configuration := config.Config{OrganizationId: "org_1", ProjectId: &projectID, ProjectAlternateId: &projectSlug, AppAlternateId: &appSlug}
			service := &BuildService{baseUrl: server.URL, httpClient: client}
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			uris, ports, err := service.buildDocker(cmd, cprint.NewCPrinter(false), configuration, Options{DockerfilePath: dir}, "build")
			if scenario == "registry-changed" || scenario == "missing-credentials" {
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "credential-")
				assert.Zero(t, manifestCalls)
				assert.Zero(t, buildCalls)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{artifact}, uris)
			assert.Equal(t, []int{80}, ports)
			_, err = service.CreateBuild(CreateBuildOptions{OrganizationId: "org_1", AppId: "app_1", DockerUris: uris, Ports: ports})
			require.NoError(t, err)
			assert.Equal(t, 1, buildCalls)
			expectedCalls := 2
			if scenario != "upload" {
				expectedCalls = 3
			}
			assert.Equal(t, expectedCalls, registryCalls)
			args, err := os.ReadFile(filepath.Join(dir, "args"))
			require.NoError(t, err)
			assert.Contains(t, string(args), registry.Url+"/shop/api:build")
			assert.NotContains(t, string(args), "credential-")
			logins, err := os.ReadFile(filepath.Join(dir, "logins"))
			require.NoError(t, err)
			assert.Contains(t, string(logins), "credential-2")
			assert.NotContains(t, string(logins), "credential-1")
		})
	}
}

func TestCloudflareDigestUsesResponseMediaType(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType string
		valid                   bool
	}{
		{"oci", `{"schemaVersion":2}`, "application/vnd.oci.image.manifest.v1+json; charset=utf-8", true},
		{"docker", `{"schemaVersion":2,"mediaType":""}`, "application/vnd.docker.distribution.manifest.v2+json", true},
		{"missing", `{"schemaVersion":2}`, "", false},
		{"unknown-header", `{"schemaVersion":2}`, "application/json", false},
		{"explicit-unknown", `{"schemaVersion":2,"mediaType":"unsupported"}`, "application/vnd.oci.image.manifest.v1+json", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := new(httputil.MockHTTPClient)
			client.On("Do", mock.Anything).Return(&http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {tc.contentType}},
				Body:       io.NopCloser(strings.NewReader(tc.body)),
			}, nil).Once()
			registry := registryFixture()
			uri, err := resolveCloudflareDigest(context.Background(), client, registry.Url+"/shop/api:build", registry)
			if tc.valid {
				require.NoError(t, err)
				sum := sha256.Sum256([]byte(tc.body))
				assert.Equal(t, registry.Url+"/shop/api@sha256:"+hex.EncodeToString(sum[:]), uri)
			} else {
				require.ErrorContains(t, err, "unsupported Cloudflare image manifest")
			}
			client.AssertExpectations(t)
		})
	}
}

func TestCloudflareIndexUsesResponseMediaType(t *testing.T) {
	manifest := `{"schemaVersion":2}`
	sum := sha256.Sum256([]byte(manifest))
	digest := "sha256:" + hex.EncodeToString(sum[:])
	index := fmt.Sprintf(`{"schemaVersion":2,"manifests":[{"digest":%q,"platform":{"os":"linux","architecture":"amd64"}}]}`, digest)
	for _, mismatch := range []bool{false, true} {
		t.Run(fmt.Sprint(mismatch), func(t *testing.T) {
			client := new(httputil.MockHTTPClient)
			client.On("Do", mock.Anything).Return(&http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/vnd.oci.image.index.v1+json"}},
				Body:       io.NopCloser(strings.NewReader(index)),
			}, nil).Once()
			body := manifest
			if mismatch {
				body += " "
			}
			client.On("Do", mock.Anything).Run(func(args mock.Arguments) {
				assert.True(t, strings.HasSuffix(args.Get(0).(*http.Request).URL.Path, "/manifests/"+digest))
			}).Return(&http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/vnd.oci.image.manifest.v1+json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil).Once()
			registry := registryFixture()
			uri, err := resolveCloudflareDigest(context.Background(), client, registry.Url+"/shop/api:build", registry)
			if mismatch {
				require.ErrorContains(t, err, "manifest digest mismatch")
			} else {
				require.NoError(t, err)
				assert.Equal(t, registry.Url+"/shop/api@"+digest, uri)
			}
			client.AssertExpectations(t)
		})
	}
}

type manifestErrorBody struct {
	read, closed bool
}

func (b *manifestErrorBody) Read([]byte) (int, error) {
	b.read = true
	return 0, errors.New("credential-sentinel")
}

func (b *manifestErrorBody) Close() error {
	b.closed = true
	return nil
}

func TestCloudflareDigestClosesErrorResponsesWithoutReading(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			body := &manifestErrorBody{}
			client := new(httputil.MockHTTPClient)
			client.On("Do", mock.Anything).Return(&http.Response{StatusCode: status, Body: body}, nil).Once()
			registry := registryFixture()
			_, err := resolveCloudflareDigest(context.Background(), client, registry.Url+"/shop/api:build", registry)
			require.Error(t, err)
			if status == http.StatusUnauthorized {
				assert.ErrorIs(t, err, dockerutil.ErrRegistryAuthentication)
			} else {
				assert.ErrorContains(t, err, "Cloudflare manifest is unavailable")
			}
			assert.NotContains(t, err.Error(), "credential-sentinel")
			assert.False(t, body.read)
			assert.True(t, body.closed)
			client.AssertExpectations(t)
		})
	}
}
