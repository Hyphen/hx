package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deployConfigFS struct{ fsutil.FileSystem }

func (deployConfigFS) ReadFile(string) ([]byte, error) {
	return []byte(`{"organization_id":"org_test","project_id":"proj_test","app_id":"app_web","app_alternate_id":"website","hyphen_api_key":"apix-test-key"}`), nil
}

type deployTransport func(*http.Request) (*http.Response, error)

func (transport deployTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func TestRunDeployWithStaticSite(t *testing.T) {
	for _, skipBuild := range []bool{true, false} {
		t.Run(fmt.Sprint(skipBuild), func(t *testing.T) {
			oldFS, oldTransport, oldPrinter := config.FS, http.DefaultTransport, printer
			oldNoBuild, oldType, oldApps, oldSites := noBuild, buildType, appsFlag, sitesFlag
			oldOrg, oldPreview, oldPrefix := flags.OrganizationFlag, flags.PreviewNameFlag, flags.PreviewPrefixFlag
			defer func() {
				config.FS, http.DefaultTransport, printer = oldFS, oldTransport, oldPrinter
				noBuild, buildType, appsFlag, sitesFlag = oldNoBuild, oldType, oldApps, oldSites
				flags.OrganizationFlag, flags.PreviewNameFlag, flags.PreviewPrefixFlag = oldOrg, oldPreview, oldPrefix
			}()
			config.FS = deployConfigFS{oldFS}
			printer = cprint.NewCPrinter(false)
			noBuild, buildType, appsFlag, sitesFlag = skipBuild, "docker", "api:latest", "website:abld_existing"
			flags.OrganizationFlag, flags.PreviewNameFlag, flags.PreviewPrefixFlag = "org_test", "branch", ""
			args := []string{"dply_test"}
			if !skipBuild {
				buildType, sitesFlag = "static", "website"
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>hello-world</html>"), 0600))
				args = append(args, dir)
			}
			var calls []string
			// Every request stays inside this transport; no network or live APIs.
			http.DefaultTransport = deployTransport(func(req *http.Request) (*http.Response, error) {
				calls = append(calls, req.Method+" "+req.URL.Path)
				code, body := 200, `{}`
				if req.URL.Host == "uploads.test" {
					assert.Equal(t, "Bearer capability", req.Header.Get("Authorization"))
					assert.Empty(t, req.Header.Get("X-API-Key"))
				} else {
					assert.Equal(t, "apix-test-key", req.Header.Get("X-API-Key"))
				}
				switch req.URL.Path {
				case "/api/organizations/org_test/deployments/dply_test":
					body = `{"id":"dply_test","isReady":true,"project":{"id":"proj_test"},"projectEnvironment":{"id":"env_test"},"apps":[{"app":{"id":"app_api","alternateId":"api"}}],"sites":[{"app":{"id":"app_web","alternateId":"website"}}],"previews":[{"id":"prev_test","name":"branch","hostPrefix":"preview-hash"}]}`
				case "/api/organizations/org_test/integrations/connections":
					body = `{"data":[{"id":"conn_registry","type":"SiteRegistry","status":"Ready","organization":{"id":"org_test"},"project":{"id":"proj_test"},"entity":{"id":"proj_test","type":"Project"},"organizationIntegration":{"id":"oint_cloud","type":"hyphenCloud"}}]}`
				case "/api/organizations/org_test/apps/app_web/builds/uploads":
					var input map[string]any
					assert.NoError(t, json.NewDecoder(req.Body).Decode(&input))
					assert.Equal(t, "env_test", input["environmentId"])
					assert.Equal(t, "preview-hash", input["previewHash"])
					body = fmt.Sprintf(`{"sessionId":"session","revision":"abcd1234","token":"capability","expiresAt":%q,"uploadUrl":"https://uploads.test/files","finalizeUrl":"https://uploads.test/complete","abortUrl":"https://uploads.test/abort","limits":{"maxFiles":100,"maxBytes":100000,"maxFileBytes":10000}}`, time.Now().Add(time.Minute).Format(time.RFC3339))
				case "/files":
					assert.Contains(t, req.Header.Get("Content-Type"), "multipart/form-data;")
					_, err := io.Copy(io.Discard, req.Body)
					assert.NoError(t, err)
				case "/complete":
					body = `{"state":"sealed","organizationId":"org_test","projectId":"proj_test","appId":"app_web","revision":"abcd1234"}`
				case "/api/organizations/org_test/apps/app_web/builds":
					assert.Equal(t, "POST /complete", calls[len(calls)-2])
					code, body = 201, `{"id":"abld_new","app":{"id":"app_web"}}`
				case "/api/organizations/org_test/deployments/dply_test/runs":
					var input map[string]json.RawMessage
					assert.NoError(t, json.NewDecoder(req.Body).Decode(&input))
					buildID := "abld_existing"
					if !skipBuild {
						buildID = "abld_new"
					}
					assert.JSONEq(t, fmt.Sprintf(`[{"appId":"app_api","build":"latest"},{"appId":"app_web","buildId":%q}]`, buildID), string(input["artifacts"]))
					assert.JSONEq(t, `"prev_test"`, string(input["previewId"]))
					// Stop before event monitoring, after checking the real command's run payload.
					code, body = 400, `{"message":"fixture-complete"}`
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
					code = 404
				}
				return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: req}, nil
			})
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			_, err := runDeployBody(cmd, args)
			require.ErrorContains(t, err, "fixture-complete")
			assert.Equal(t, "POST /api/organizations/org_test/deployments/dply_test/runs", calls[len(calls)-1])
			if skipBuild {
				assert.Len(t, calls, 2)
			} else {
				assert.Contains(t, calls, "POST /complete")
			}
		})
	}
}
