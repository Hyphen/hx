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
		name, selector, environment, wantPath string
	}{
		{"deployment ID", "depl_123", "", "/deployments/depl_123"},
		{"deployment alternate ID", "web-development", "", "/deployments/web-development"},
		{"explicit environment alternate ID", "", "production", "/projects/proj_1/environments/production/deployment"},
		{"explicit environment ID", "", "env_prod", "/projects/proj_1/environments/env_prod/deployment"},
		{"default development type", "", "", "/projects/proj_1/environments/env_dev/deployment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			originalFS, originalTransport := config.FS, http.DefaultTransport
			originalEnv, originalProject, originalApps, originalOrg, originalPrinter := envFlag, projectFlag, appsFlag, flags.OrganizationFlag, printer
			t.Cleanup(func() {
				config.FS, http.DefaultTransport = originalFS, originalTransport
				envFlag, projectFlag, appsFlag, flags.OrganizationFlag, printer = originalEnv, originalProject, originalApps, originalOrg, originalPrinter
			})
			config.FS = &fsutil.MockFileSystem{ReadFileFunc: func(path string) ([]byte, error) {
				if path != config.ManifestConfigFile {
					return nil, os.ErrNotExist
				}
				return []byte(`{"organization_id":"org_1","project_id":"proj_1","app_id":"app_1","hyphen_api_key":"test-only"}`), nil
			}}
			envFlag, projectFlag, appsFlag, flags.OrganizationFlag = tc.environment, "", "", "org_1"
			printer = cprint.NewCPrinter(false)
			client := new(httputil.MockHTTPClient)
			http.DefaultTransport = selectionTransport{client}
			if tc.selector == "" && tc.environment == "" {
				client.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.Method == http.MethodGet && req.URL.Path == "/api/organizations/org_1/projects/proj_1/environments"
				})).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"data":[{"id":"env_prod","alternateId":"development","type":"production"},{"id":"env_dev","alternateId":"sandbox","type":"development"}]}`,
				))}, nil).Once()
			}
			client.On("Do", mock.MatchedBy(func(req *http.Request) bool {
				return req.Method == http.MethodGet && req.URL.Path == "/api/organizations/org_1"+tc.wantPath
			})).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"id":"depl_selected","isReady":false,"apps":[{"app":{"id":"app_1"}}],"readinessIssues":[{"error":"ContainerRegistry connection not found"}]}`,
			))}, nil).Once()
			var args []string
			if tc.selector != "" {
				args = []string{tc.selector}
			}
			result, err := runDeployBody(DeployCmd, args)
			require.ErrorContains(t, err, "deployment not ready: ContainerRegistry connection not found")
			assert.Equal(t, "depl_selected", result["deploymentId"])
			client.AssertExpectations(t)
		})
	}
}
