package appcloud

import "github.com/stretchr/testify/mock"

// MockAppCloudService is a testify mock of AppCloudServicer.
type MockAppCloudService struct {
	mock.Mock
}

var _ AppCloudServicer = (*MockAppCloudService)(nil)

func (m *MockAppCloudService) ListApps(organizationID, projectID string) ([]App, error) {
	args := m.Called(organizationID, projectID)
	return args.Get(0).([]App), args.Error(1)
}

func (m *MockAppCloudService) GetApp(organizationID, appID string) (App, error) {
	args := m.Called(organizationID, appID)
	return args.Get(0).(App), args.Error(1)
}
