package appcloud

import (
	"io"

	"github.com/stretchr/testify/mock"
)

// MockAppCloudService is a testify mock of AppCloudServicer.
type MockAppCloudService struct {
	mock.Mock
}

var _ AppCloudServicer = (*MockAppCloudService)(nil)

func (m *MockAppCloudService) ListApps(organizationID, projectID string) ([]App, error) {
	args := m.Called(organizationID, projectID)
	return args.Get(0).([]App), args.Error(1)
}

func (m *MockAppCloudService) GetApp(appID string) (App, error) {
	args := m.Called(appID)
	return args.Get(0).(App), args.Error(1)
}

func (m *MockAppCloudService) FindAppByDomain(domain string) (*App, error) {
	args := m.Called(domain)
	app, _ := args.Get(0).(*App)
	return app, args.Error(1)
}

func (m *MockAppCloudService) CreateApp(owner, name, organizationID, projectID string, domains []string) (App, error) {
	args := m.Called(owner, name, organizationID, projectID, domains)
	return args.Get(0).(App), args.Error(1)
}

func (m *MockAppCloudService) DeleteApp(appID string) error {
	args := m.Called(appID)
	return args.Error(0)
}

func (m *MockAppCloudService) CreateRevision(appID, kind, artifactRef string) (Revision, error) {
	args := m.Called(appID, kind, artifactRef)
	return args.Get(0).(Revision), args.Error(1)
}

func (m *MockAppCloudService) UploadBatch(appID, hex string, body io.Reader, contentType string) (UploadResponse, error) {
	args := m.Called(appID, hex, body, contentType)
	return args.Get(0).(UploadResponse), args.Error(1)
}

func (m *MockAppCloudService) SetActiveRevision(appID, hex string) (App, error) {
	args := m.Called(appID, hex)
	return args.Get(0).(App), args.Error(1)
}

func (m *MockAppCloudService) GetConfig(appID string) (ConfigView, error) {
	args := m.Called(appID)
	return args.Get(0).(ConfigView), args.Error(1)
}

func (m *MockAppCloudService) SetConfig(appID string, config AppConfig, ifMatch string) (ConfigView, error) {
	args := m.Called(appID, config, ifMatch)
	return args.Get(0).(ConfigView), args.Error(1)
}

func (m *MockAppCloudService) QueryHTTPMetrics(params HTTPMetricsParams) ([]HTTPLog, error) {
	args := m.Called(params)
	return args.Get(0).([]HTTPLog), args.Error(1)
}

func (m *MockAppCloudService) QueryErrorMetrics(params ErrorMetricsParams) ([]ErrorLog, error) {
	args := m.Called(params)
	return args.Get(0).([]ErrorLog), args.Error(1)
}
