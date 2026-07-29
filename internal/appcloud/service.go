package appcloud

import (
	"bytes"
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
type AppCloudServicer interface {
	ListApps(organizationID, projectID string) ([]App, error)
	GetApp(appID string) (App, error)
	FindAppByDomain(domain string) (*App, error)
	CreateApp(owner, name, organizationID, projectID string, domains []string) (App, error)
	DeleteApp(appID string) error
	CreateRevision(appID, kind, artifactRef string) (Revision, error)
	UploadBatch(appID, hex string, body io.Reader, contentType string) (UploadResponse, error)
	SetActiveRevision(appID, hex string) (App, error)
	GetConfig(appID string) (ConfigView, error)
	SetConfig(appID string, config AppConfig, ifMatch string) (ConfigView, error)
	QueryHTTPMetrics(params HTTPMetricsParams) ([]HTTPLog, error)
	QueryErrorMetrics(params ErrorMetricsParams) ([]ErrorLog, error)
}

// AppCloudService talks to the AppCloud management control plane. Auth is
// handled by the shared Hyphen HTTP client (OAuth bearer / x-api-key); the
// base URL follows the standard dev/local/prod switch.
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

// do issues a JSON request. `payload` (if non-nil) is marshalled as the body;
// `out` (if non-nil) is unmarshalled from a successful response. `okStatus` is
// the expected success code.
func (s *AppCloudService) do(method, path string, payload, out any, okStatus int) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return errors.Wrap(err, "Failed to marshal request payload")
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.baseUrl+path, body)
	if err != nil {
		return errors.Wrap(err, "Failed to create request")
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != okStatus {
		return errors.HandleHTTPError(resp)
	}
	if out == nil {
		return nil
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Failed to read response body")
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return errors.Wrap(err, "Failed to parse JSON response")
	}
	return nil
}

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
		var page listResponse[App]
		if err := s.do(http.MethodGet, "/v1/apps?"+q.Encode(), nil, &page, http.StatusOK); err != nil {
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

func (s *AppCloudService) GetApp(appID string) (App, error) {
	var app App
	err := s.do(http.MethodGet, "/v1/apps/"+url.PathEscape(appID), nil, &app, http.StatusOK)
	return app, err
}

// FindAppByDomain returns the app that owns `domain`, or nil if none. Domain
// uniqueness is a server invariant, so >1 match is a hard error.
func (s *AppCloudService) FindAppByDomain(domain string) (*App, error) {
	q := url.Values{}
	q.Set("domain", domain)
	var page listResponse[App]
	if err := s.do(http.MethodGet, "/v1/apps?"+q.Encode(), nil, &page, http.StatusOK); err != nil {
		return nil, err
	}
	switch len(page.Items) {
	case 0:
		return nil, nil
	case 1:
		return &page.Items[0], nil
	default:
		return nil, fmt.Errorf("management returned %d apps for domain %s; expected 0 or 1", len(page.Items), domain)
	}
}

func (s *AppCloudService) CreateApp(owner, name, organizationID, projectID string, domains []string) (App, error) {
	if domains == nil {
		domains = []string{}
	}
	payload := map[string]any{
		"owner":          owner,
		"name":           name,
		"organizationId": organizationID,
		"projectId":      projectID,
		"domains":        domains,
		"config":         map[string]any{},
	}
	var app App
	err := s.do(http.MethodPost, "/v1/apps", payload, &app, http.StatusCreated)
	return app, err
}

func (s *AppCloudService) DeleteApp(appID string) error {
	return s.do(http.MethodDelete, "/v1/apps/"+url.PathEscape(appID), nil, nil, http.StatusNoContent)
}

func (s *AppCloudService) CreateRevision(appID, kind, artifactRef string) (Revision, error) {
	payload := map[string]any{"kind": kind, "artifactRef": artifactRef}
	var rev Revision
	err := s.do(http.MethodPost, "/v1/apps/"+url.PathEscape(appID)+"/revisions", payload, &rev, http.StatusCreated)
	return rev, err
}

// UploadBatch POSTs a pre-built multipart body (see upload.go) of gzipped
// files for a revision.
func (s *AppCloudService) UploadBatch(appID, hex string, body io.Reader, contentType string) (UploadResponse, error) {
	var out UploadResponse
	path := fmt.Sprintf("/v1/apps/%s/revisions/%s/files", url.PathEscape(appID), url.PathEscape(hex))
	req, err := http.NewRequest(http.MethodPost, s.baseUrl+path, body)
	if err != nil {
		return out, errors.Wrap(err, "Failed to create request")
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	// The upload endpoint returns 201 Created; accept any 2xx.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, errors.HandleHTTPError(resp)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return out, errors.Wrap(err, "Failed to read response body")
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, errors.Wrap(err, "Failed to parse JSON response")
	}
	return out, nil
}

func (s *AppCloudService) SetActiveRevision(appID, hex string) (App, error) {
	payload := map[string]any{"activeRevision": hex}
	var app App
	err := s.do(http.MethodPatch, "/v1/apps/"+url.PathEscape(appID), payload, &app, http.StatusOK)
	return app, err
}

func (s *AppCloudService) GetConfig(appID string) (ConfigView, error) {
	var cv ConfigView
	err := s.do(http.MethodGet, "/v1/apps/"+url.PathEscape(appID)+"/config", nil, &cv, http.StatusOK)
	return cv, err
}

// SetConfig replaces the app's config. `ifMatch` (if non-empty) sends an
// If-Match header for optimistic concurrency (412 on mismatch).
func (s *AppCloudService) SetConfig(appID string, config AppConfig, ifMatch string) (ConfigView, error) {
	var cv ConfigView
	b, err := json.Marshal(config)
	if err != nil {
		return cv, errors.Wrap(err, "Failed to marshal config")
	}
	req, err := http.NewRequest(http.MethodPut, s.baseUrl+"/v1/apps/"+url.PathEscape(appID)+"/config", bytes.NewReader(b))
	if err != nil {
		return cv, errors.Wrap(err, "Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return cv, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return cv, errors.HandleHTTPError(resp)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return cv, errors.Wrap(err, "Failed to read response body")
	}
	if err := json.Unmarshal(raw, &cv); err != nil {
		return cv, errors.Wrap(err, "Failed to parse JSON response")
	}
	return cv, nil
}

// HTTPMetricsParams are the filters for `GET /v1/metrics/http`. Empty fields
// are omitted so the server defaults apply.
type HTTPMetricsParams struct {
	AppID  string
	Domain string
	Method string
	Status int
	From   string
	To     string
	Limit  int
}

// ErrorMetricsParams are the filters for `GET /v1/metrics/errors`.
type ErrorMetricsParams struct {
	AppID  string
	Domain string
	Kind   string
	From   string
	To     string
	Limit  int
}

func (s *AppCloudService) QueryHTTPMetrics(params HTTPMetricsParams) ([]HTTPLog, error) {
	q := url.Values{}
	setStr(q, "appId", params.AppID)
	setStr(q, "domain", params.Domain)
	setStr(q, "method", params.Method)
	setInt(q, "status", params.Status)
	setStr(q, "from", params.From)
	setStr(q, "to", params.To)
	setInt(q, "limit", params.Limit)
	var page listResponse[HTTPLog]
	err := s.do(http.MethodGet, "/v1/metrics/http?"+q.Encode(), nil, &page, http.StatusOK)
	return page.Items, err
}

func (s *AppCloudService) QueryErrorMetrics(params ErrorMetricsParams) ([]ErrorLog, error) {
	q := url.Values{}
	setStr(q, "appId", params.AppID)
	setStr(q, "domain", params.Domain)
	setStr(q, "kind", params.Kind)
	setStr(q, "from", params.From)
	setStr(q, "to", params.To)
	setInt(q, "limit", params.Limit)
	var page listResponse[ErrorLog]
	err := s.do(http.MethodGet, "/v1/metrics/errors?"+q.Encode(), nil, &page, http.StatusOK)
	return page.Items, err
}

func setStr(q url.Values, key, val string) {
	if val != "" {
		q.Set(key, val)
	}
}

func setInt(q url.Values, key string, val int) {
	if val != 0 {
		q.Set(key, fmt.Sprintf("%d", val))
	}
}
