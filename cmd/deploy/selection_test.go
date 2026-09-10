package deploy

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/Hyphen/cli/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type selectionTransport struct{ client *httputil.MockHTTPClient }

func (transport selectionTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport.client.Do(request)
}

func TestDeploymentSelection(t *testing.T) {
	for _, tc := range []struct {
		name                                        string
		args                                        []string
		globalProject, developmentProject, wantPath string
	}{
		{"development type in configured project", nil, "", "proj_1", "/projects/proj_1/environments/env_dev/deployment"},
		{"explicit environment alternate ID", []string{"--env", "production"}, "", "", "/projects/proj_1/environments/production/deployment"},
		{"explicit environment ID", []string{"--env", "env_prod"}, "", "", "/projects/proj_1/environments/env_prod/deployment"},
		{"explicit project overrides local config", []string{"--project", "web", "--env", "production"}, "", "", "/projects/web/environments/production/deployment"},
		{"project shorthand overrides local config", []string{"-p", "web", "--env", "production"}, "", "", "/projects/web/environments/production/deployment"},
		{"development type in explicit project", []string{"--project", "web"}, "", "web", "/projects/web/environments/env_dev/deployment"},
		{"project set on root command overrides local config", []string{"--env", "production"}, "web", "", "/projects/web/environments/production/deployment"},
		{"deployment ID compatibility", []string{"depl_123", "--project", "web", "--env", "production"}, "", "", "/deployments/depl_123"},
		{"deployment alternate ID compatibility", []string{"web-development"}, "", "", "/deployments/web-development"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			originalFS, originalTransport := config.FS, http.DefaultTransport
			originalEnv, originalProject, originalApps, originalOrg, originalPrinter := envFlag, flags.ProjectFlag, appsFlag, flags.OrganizationFlag, printer
			t.Cleanup(func() {
				config.FS, http.DefaultTransport = originalFS, originalTransport
				envFlag, flags.ProjectFlag, appsFlag, flags.OrganizationFlag, printer = originalEnv, originalProject, originalApps, originalOrg, originalPrinter
			})
			config.FS = &fsutil.MockFileSystem{ReadFileFunc: func(path string) ([]byte, error) {
				if path != config.ManifestConfigFile {
					return nil, os.ErrNotExist
				}
				return []byte(`{"organization_id":"org_1","project_id":"proj_1","app_id":"app_1","hyphen_api_key":"test-only"}`), nil
			}}
			envFlag, flags.ProjectFlag, appsFlag, flags.OrganizationFlag = "", tc.globalProject, "", "org_1"
			printer = cprint.NewCPrinter(false)
			client := new(httputil.MockHTTPClient)
			http.DefaultTransport = selectionTransport{client}
			if tc.developmentProject != "" {
				client.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.Method == http.MethodGet && req.URL.Path == "/api/organizations/org_1/projects/"+tc.developmentProject+"/environments"
				})).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"data":[{"id":"env_prod","alternateId":"development","type":"production"},{"id":"env_dev","alternateId":"sandbox","type":"development"}]}`,
				))}, nil).Once()
			}
			client.On("Do", mock.MatchedBy(func(req *http.Request) bool {
				return req.Method == http.MethodGet && req.URL.Path == "/api/organizations/org_1"+tc.wantPath
			})).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"id":"depl_selected","isReady":false,"apps":[{"app":{"id":"app_1"}}],"readinessIssues":[{"error":"ContainerRegistry connection not found"}]}`,
			))}, nil).Once()
			require.NoError(t, DeployCmd.ParseFlags(tc.args))
			args := DeployCmd.Flags().Args()
			require.NoError(t, DeployCmd.Args(DeployCmd, args))
			result, err := runDeployBody(DeployCmd, args)
			require.ErrorContains(t, err, "deployment not ready: ContainerRegistry connection not found")
			assert.Equal(t, "depl_selected", result["deploymentId"])
			client.AssertExpectations(t)
		})
	}
}
