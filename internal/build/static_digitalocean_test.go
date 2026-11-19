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

	"github.com/Hyphen/cli/internal/models"
	"github.com/stretchr/testify/require"
)

func TestDigitalOceanStaticImagePush(t *testing.T) {
	for _, scenario := range []string{"success", "expired", "changed", "push failure"} {
		t.Run(scenario, func(t *testing.T) {
			directory := t.TempDir()
			site := filepath.Join(directory, "source")
			require.NoError(t, os.MkdirAll(filepath.Join(site, "assets"), 0755))
			require.NoError(t, os.WriteFile(filepath.Join(site, "index.html"), []byte("<h1>snapshot</h1>"), 0644))
			require.NoError(t, os.WriteFile(filepath.Join(site, "assets", "empty.txt"), nil, 0644))
			files, err := prepareSite(site, directory)
			require.NoError(t, err)
			// The image must use the hashed snapshot even if the source changes after preparation.
			require.NoError(t, os.WriteFile(filepath.Join(site, "index.html"), []byte("changed"), 0644))
			digest := "sha256:" + strings.Repeat("a", 64)
			script := `#!/bin/sh
printf '%s\n' "$@" >> "$DOCKER_TEST_DIR/args"
if [ "$1" = build ]; then
 for argument in "$@"; do build_context="$argument"; done
 /bin/cp "$build_context/Dockerfile" "$DOCKER_TEST_DIR/built-dockerfile"
 /bin/cp "$build_context/site/index.html" "$DOCKER_TEST_DIR/built-index"
 test -f "$build_context/site/assets/empty.txt" || exit 1
fi
if [ "$3" = login ]; then /bin/cat >> "$DOCKER_TEST_DIR/logins"; fi
if [ "$3" = push ]; then
 if [ "$SCENARIO" = "push failure" ]; then echo secret-sentinel >&2; exit 1; fi
 if [ "$SCENARIO" = expired ] && [ ! -f "$DOCKER_TEST_DIR/failed" ]; then
  : > "$DOCKER_TEST_DIR/failed"
  echo unauthorized >&2
  exit 1
 fi
 echo 'build: digest: ` + digest + ` size: 123'
fi
`
			require.NoError(t, os.WriteFile(filepath.Join(directory, "docker"), []byte(script), 0700))
			t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("DOCKER_TEST_DIR", directory)
			t.Setenv("SCENARIO", scenario)
			registry := models.ContainerRegistry{Id: "conn_site", Name: "team", Url: "registry.digitalocean.com/team/static-project"}
			capability := uploadCapability{Provider: "digitalOcean", AppID: "app_1", Registry: registry}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/organizations/org_1/apps/app_1/builds/uploads", r.URL.Path)
				calls++
				response := capability
				response.Registry.Auth.Server = "registry.digitalocean.com"
				response.Registry.Auth.Username = "user"
				response.Registry.Auth.Password = fmt.Sprintf("fresh-sentinel-%d", calls)
				if scenario == "changed" {
					response.Registry.Url += "-other"
				}
				require.NoError(t, json.NewEncoder(w).Encode(response))
			}))
			defer server.Close()
			service := &BuildService{baseUrl: server.URL, httpClient: server.Client()}
			artifact, err := service.uploadDigitalOceanSite(context.Background(), "org_1", "app_1", "web", "conn_site", files, capability, Options{})
			if scenario == "changed" || scenario == "push failure" {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "sentinel")
				require.Nil(t, artifact)
			} else {
				require.NoError(t, err)
				require.Equal(t, &models.StaticArtifact{Provider: "digitalOcean", RegistryID: "conn_site", URI: registry.Url + "/web@" + digest}, artifact)
				logins, err := os.ReadFile(filepath.Join(directory, "logins"))
				require.NoError(t, err)
				require.Contains(t, string(logins), "fresh-sentinel-1")
			}
			expectedCalls := 1
			if scenario == "expired" {
				expectedCalls = 2
			}
			require.Equal(t, expectedCalls, calls)
			args, err := os.ReadFile(filepath.Join(directory, "args"))
			require.NoError(t, err)
			require.Contains(t, string(args), "--platform\nlinux/amd64")
			require.NotContains(t, string(args), "sentinel")
			dockerfile, err := os.ReadFile(filepath.Join(directory, "built-dockerfile"))
			require.NoError(t, err)
			require.Equal(t, "FROM scratch\nCOPY site/ /site/\n", string(dockerfile))
			index, err := os.ReadFile(filepath.Join(directory, "built-index"))
			require.NoError(t, err)
			require.Equal(t, "<h1>snapshot</h1>", string(index))
		})
	}
}

func TestDigitalOceanRejectsControlFilesAndCorruptSnapshots(t *testing.T) {
	directory := t.TempDir()
	site := filepath.Join(directory, "source")
	require.NoError(t, os.MkdirAll(site, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(site, "_headers"), []byte("unsupported"), 0644))
	files, err := prepareSite(site, directory)
	require.NoError(t, err)
	_, err = stageStaticImage(t.TempDir(), files)
	require.ErrorContains(t, err, "control files")
	files[0].Path = "index.html"
	files[0].SHA256 = strings.Repeat("0", 64)
	_, err = stageStaticImage(t.TempDir(), files)
	require.ErrorContains(t, err, "changed")
}
