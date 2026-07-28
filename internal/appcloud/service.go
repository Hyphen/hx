package appcloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

// AppCloudServicer is the AppCloud management API surface the CLI depends on.
// Kept small and interface-first so commands can be tested against a mock.
type AppCloudServicer interface {
	ListApps(organizationID, projectID string) ([]App, error)
	GetApp(organizationID, appID string) (App, error)
}

// AppCloudService talks to the AppCloud management control plane. Auth is
// handled by the shared Hyphen HTTP client, which attaches the caller's OAuth
// bearer token; the base URL follows the standard dev/local/prod switch.
type AppCloudService struct {
	baseUrl    string
	httpClient httputil.Client
}

var _ AppCloudServicer = (*AppCloudService)(nil)

func NewService() *AppCloudService {
	return &AppCloudService{
		baseUrl:    apiconf.GetBaseAppCloudUrl(),
		httpClient: httputil.NewHyphenHTTPClient(),
	}
}

// ListApps returns the apps visible to the caller in an organization,
// optionally narrowed to a project. Follows the cursor to return all pages.
func (s *AppCloudService) ListApps(organizationID, projectID string) ([]App, error) {
	var apps []App
	cursor := ""
	for {
		q := url.Values{}
		q.Set("organizationId", organizationID)
		if projectID != "" {
			q.Set("projectId", projectID)
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		endpoint := fmt.Sprintf("%s/v1/apps?%s", s.baseUrl, q.Encode())

		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create request")
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		page, err := decodeListResponse(resp)
		if err != nil {
			return nil, err
		}

		apps = append(apps, page.Items...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return apps, nil
}

// GetApp fetches a single app by id (the management API scopes it to the
// caller's org membership; a hidden app surfaces as not found).
func (s *AppCloudService) GetApp(organizationID, appID string) (App, error) {
	endpoint := fmt.Sprintf("%s/v1/apps/%s", s.baseUrl, url.PathEscape(appID))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return App{}, errors.Wrap(err, "Failed to create request")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return App{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return App{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return App{}, errors.Wrap(err, "Failed to read response body")
	}

	var app App
	if err := json.Unmarshal(body, &app); err != nil {
		return App{}, errors.Wrap(err, "Failed to parse JSON response")
	}
	return app, nil
}

func decodeListResponse(resp *http.Response) (listResponse, error) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return listResponse{}, errors.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return listResponse{}, errors.Wrap(err, "Failed to read response body")
	}

	var page listResponse
	if err := json.Unmarshal(body, &page); err != nil {
		return listResponse{}, errors.Wrap(err, "Failed to parse JSON response")
	}
	return page, nil
}
