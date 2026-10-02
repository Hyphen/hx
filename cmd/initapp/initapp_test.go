package initapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/fsutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type initTransport func(*http.Request) (*http.Response, error)

func (transport initTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func TestRunInitAppUsesAppProject(t *testing.T) {
	for _, tc := range []struct {
		name      string
		existing  bool
		projectID string
	}{
		{name: "existing app in another project", existing: true, projectID: "proj_app"},
		{name: "new app in default project", projectID: "proj_default"},
		{name: "existing app without a project", existing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			oldFS, oldTransport, oldPrinter := config.FS, http.DefaultTransport, printer
			oldOrg, oldProject, oldLocalSecret, oldAppID := flags.OrganizationFlag, flags.ProjectFlag, flags.LocalSecret, appIDFlag
			t.Cleanup(func() {
				config.FS, http.DefaultTransport, printer = oldFS, oldTransport, oldPrinter
				flags.OrganizationFlag, flags.ProjectFlag, flags.LocalSecret, appIDFlag = oldOrg, oldProject, oldLocalSecret, oldAppID
			})
			flags.OrganizationFlag, flags.ProjectFlag, flags.LocalSecret, appIDFlag = "", "", false, ""

			globalPath := filepath.Join(config.GetGlobalDirectory(), config.ManifestConfigFile)
			globalConfig := []byte(`{"organization_id":"org_test","project_id":"proj_default","project_name":"Default Project","project_alternate_id":"default-project","hyphen_api_key":"test-key"}`)
			files := map[string][]byte{globalPath: globalConfig}
			config.FS = &fsutil.MockFileSystem{
				ReadFileFunc: func(path string) ([]byte, error) {
					if data, ok := files[path]; ok {
						return data, nil
					}
					return nil, os.ErrNotExist
				},
				WriteFileFunc: func(path string, data []byte, _ os.FileMode) error {
					require.Equal(t, config.ManifestConfigFile, path)
					files[path] = data
					return nil
				},
				StatFunc: func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
			}

			selectedApp := models.App{
				ID: "app_test", Name: "Matching App", AlternateId: "matching-app",
				Project: models.ProjectReference{ID: tc.projectID, Name: "App Project", AlternateID: "app-project"},
			}
			appJSON, err := json.Marshal(selectedApp)
			require.NoError(t, err)
			var requests []string
			stopAtEnvironments := errors.New("environment lookup reached")
			// All HTTP requests are handled in memory; none can reach a live service.
			http.DefaultTransport = initTransport(func(req *http.Request) (*http.Response, error) {
				request := req.Method + " " + req.URL.Path
				requests = append(requests, request)
				status, body := http.StatusOK, string(appJSON)
				switch request {
				case "POST /api/organizations/org_test/projects/proj_default/apps":
					status = http.StatusCreated
					if tc.existing {
						status, body = http.StatusConflict, `{}`
					}
				case "GET /api/organizations/org_test/apps/matching-app/":
					assert.True(t, tc.existing)
				case "GET /org_test/" + tc.projectID + "/key":
					body = `{"key":{"secret_key_id":1,"secret_key":"test-secret"}}`
				case "GET /api/organizations/org_test/projects/" + tc.projectID + "/environments":
					// Stop before creating environment files or writing to the local database.
					return nil, stopAtEnvironments
				default:
					t.Errorf("unexpected request: %s", request)
					return nil, fmt.Errorf("unexpected request: %s", request)
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: req}, nil
			})

			cmd := &cobra.Command{}
			cmd.Flags().Bool("yes", true, "")
			err = RunInitAppE(cmd, []string{"Matching App"})
			assert.Equal(t, globalConfig, files[globalPath], "global defaults must stay unchanged")
			if tc.projectID == "" {
				require.ErrorContains(t, err, "has no project ID")
				assert.NotContains(t, files, config.ManifestConfigFile)
				assert.Len(t, requests, 2)
				return
			}
			assert.ErrorIs(t, err, stopAtEnvironments)
			var local config.Config
			require.NoError(t, json.Unmarshal(files[config.ManifestConfigFile], &local))
			assert.Equal(t, &tc.projectID, local.ProjectId)
			assert.Equal(t, &selectedApp.Project.Name, local.ProjectName)
			assert.Equal(t, &selectedApp.Project.AlternateID, local.ProjectAlternateId)
			assert.Equal(t, &selectedApp.ID, local.AppId)
			assert.Equal(t, &selectedApp.AlternateId, local.AppAlternateId)
			assert.Equal(t, "org_test", local.OrganizationId)
			assert.Contains(t, requests, "GET /org_test/"+tc.projectID+"/key")
			assert.Contains(t, requests, "GET /api/organizations/org_test/projects/"+tc.projectID+"/environments")
		})
	}
}
