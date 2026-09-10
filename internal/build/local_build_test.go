package build

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLocalBuildWithoutGitHub(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("The fake Docker executable uses a POSIX shell")
	}
	for _, state := range []string{"no repository", "no commits", "local commit without remote"} {
		t.Run(state, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			require.NoError(t, os.WriteFile("Dockerfile", []byte("FROM scratch\nEXPOSE 3000\n"), 0o600))
			wantSHA := "0000000"
			if state != "no repository" {
				output, err := exec.Command("git", "init", "-q").CombinedOutput()
				require.NoError(t, err, string(output))
				if state == "local commit without remote" {
					output, err = exec.Command("git", "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "Test build").CombinedOutput()
					require.NoError(t, err, string(output))
					output, err = exec.Command("git", "rev-parse", "--short=7", "HEAD").Output()
					require.NoError(t, err)
					wantSHA = strings.TrimSpace(string(output))
				}
			}
			bin := filepath.Join(dir, "bin")
			require.NoError(t, os.Mkdir(bin, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "docker"), []byte(`#!/bin/sh
printf '%s\n' "$*" >> docker-calls.txt
case "$1" in
  inspect) printf '%s\n' '[{"Config":{"ExposedPorts":{"3000/tcp":{}}}}]' ;;
  build|tag|push|login|logout) ;;
  *) exit 1 ;;
esac
`), 0o700))
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			originalFS := config.FS
			t.Cleanup(func() { config.FS = originalFS })
			config.FS = &fsutil.MockFileSystem{ReadFileFunc: func(path string) ([]byte, error) {
				if path != config.ManifestConfigFile {
					return nil, os.ErrNotExist
				}
				return []byte(`{"organization_id":"org_1","project_id":"proj_1","app_id":"app_1","app_alternate_id":"web"}`), nil
			}}
			client := new(httputil.MockHTTPClient)
			service := &BuildService{baseUrl: "https://api.example.invalid", httpClient: client}
			client.On("Do", mock.MatchedBy(func(req *http.Request) bool {
				return req.Method == http.MethodGet && req.URL.Path == "/api/organizations/org_1/deployments/containerRegistries"
			})).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`[{"id":"reg_1","url":"registry.example.invalid/images","auth":{"server":"registry.example.invalid","username":"test","password":"test"}}]`,
			))}, nil).Once()
			var metadata models.NewBuild
			client.On("Do", mock.MatchedBy(func(req *http.Request) bool {
				return req.Method == http.MethodPost && req.URL.Path == "/api/organizations/org_1/apps/app_1/builds" && req.URL.Query().Get("environmentId") == "env_dev"
			})).Run(func(args mock.Arguments) {
				require.NoError(t, json.NewDecoder(args.Get(0).(*http.Request).Body).Decode(&metadata))
			}).Return(&http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"build_1"}`))}, nil).Once()
			cmd := &cobra.Command{}
			cmd.Flags().String("registry", "", "")
			result, err := service.RunBuild(cmd, cprint.NewCPrinter(false), "env_dev", false, "Dockerfile", "")
			require.NoError(t, err)
			assert.Equal(t, "build_1", result.Id)
			assert.Equal(t, wantSHA, metadata.CommitSha)
			assert.Empty(t, metadata.CommitShaHref)
			assert.Empty(t, metadata.TagHref)
			assert.Equal(t, []int{3000}, metadata.Artifact.Ports)
			assert.Equal(t, "registry.example.invalid/images/web:"+wantSHA, metadata.Artifact.Image.URI)
			calls, err := os.ReadFile("docker-calls.txt")
			require.NoError(t, err)
			assert.Contains(t, string(calls), "build --platform linux/amd64 -f Dockerfile -t web:"+wantSHA+" .")
			assert.Contains(t, string(calls), "push "+metadata.Artifact.Image.URI)
			client.AssertExpectations(t)
		})
	}
}
