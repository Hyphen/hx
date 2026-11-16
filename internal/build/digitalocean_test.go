package build

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDigitalOceanImageDestination(t *testing.T) {
	uri, err := digitalOceanImageURI("registry.digitalocean.com/team-registry/nf-proj-abc", "api", "build")
	require.NoError(t, err)
	require.Equal(t, "registry.digitalocean.com/team-registry/nf-proj-abc/api:build", uri)
	nested, err := digitalOceanImageURI("registry.digitalocean.com/team-registry/platform/services", "api", "build")
	require.NoError(t, err)
	require.Equal(t, "registry.digitalocean.com/team-registry/platform/services/api:build", nested)
	for _, registry := range []string{"registry.digitalocean.com.evil/team/project", "https://registry.digitalocean.com/team/project", "registry.digitalocean.com/team/../project"} {
		_, err := digitalOceanImageURI(registry, "api", "build")
		require.Error(t, err)
	}
}

func TestDigitalOceanRefreshesCredentialsAndRegistersPushDigest(t *testing.T) {
	for _, scenario := range []string{"success", "expired", "changed"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			digest := "sha256:" + strings.Repeat("a", 64)
			script := `#!/bin/sh
printf '%s\n' "$@" >> "$DOCKER_TEST_DIR/args"
if [ "$3" = login ]; then /bin/cat >> "$DOCKER_TEST_DIR/logins"; fi
if [ "$3" = push ]; then
 if [ "$FAIL_PUSH" = yes ] && [ ! -f "$DOCKER_TEST_DIR/failed" ]; then
  : > "$DOCKER_TEST_DIR/failed"
  echo unauthorized >&2
  exit 1
 fi
 echo 'build: digest: ` + digest + ` size: 123'
fi
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("DOCKER_TEST_DIR", dir)
			t.Setenv("FAIL_PUSH", "no")
			if scenario == "expired" {
				t.Setenv("FAIL_PUSH", "yes")
			}
			registry := models.ContainerRegistry{Id: "conn_do", Url: "registry.digitalocean.com/team-registry/nf-proj-one"}
			registry.Auth.Server = "registry.digitalocean.com"
			registry.Auth.Username = "user"
			registry.Auth.Password = "stale-sentinel"
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/organizations/org_1/deployments/containerRegistries", r.URL.Path)
				require.Equal(t, "GET", r.Method)
				calls++
				refreshed := registry
				refreshed.Auth.Password = fmt.Sprintf("fresh-sentinel-%d", calls)
				if scenario == "changed" {
					refreshed.Url += "-changed"
				}
				json.NewEncoder(w).Encode([]models.ContainerRegistry{refreshed})
			}))
			defer server.Close()
			service := &BuildService{baseUrl: server.URL, httpClient: server.Client()}
			project, app := "proj_1", "api"
			uri, err := service.pushDigitalOcean(context.Background(), config.Config{OrganizationId: "org_1", ProjectId: &project, AppAlternateId: &app}, registry, "local:build", "build")
			if scenario == "changed" {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "sentinel")
				_, err = os.Stat(filepath.Join(dir, "args"))
				require.True(t, os.IsNotExist(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, registry.Url+"/api@"+digest, uri)
			expectedCalls := 1
			if scenario == "expired" {
				expectedCalls = 2
			}
			require.Equal(t, expectedCalls, calls)
			logins, err := os.ReadFile(filepath.Join(dir, "logins"))
			require.NoError(t, err)
			require.Contains(t, string(logins), "fresh-sentinel-1")
			require.NotContains(t, string(logins), "stale-sentinel")
			if scenario == "expired" {
				require.Contains(t, string(logins), "fresh-sentinel-2")
			}
			args, err := os.ReadFile(filepath.Join(dir, "args"))
			require.NoError(t, err)
			require.NotContains(t, string(args), "sentinel")
		})
	}
}
