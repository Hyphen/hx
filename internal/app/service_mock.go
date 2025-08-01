package app

import (
	"github.com/Hyphen/cli/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockAppService is a mock implementation of the AppServicer interface
type MockAppService struct {
	mock.Mock
}

// GetListApps mocks the GetListApps method
func (m *MockAppService) GetListApps(organizationID string, pageSize, pageNum int) ([]models.App, error) {
	args := m.Called(organizationID, pageSize, pageNum)
	return args.Get(0).([]models.App), args.Error(1)
}

// CreateApp mocks the CreateApp method
func (m *MockAppService) CreateApp(organizationID, alternateID, name string) (models.App, error) {
	args := m.Called(organizationID, alternateID, name)
	return args.Get(0).(models.App), args.Error(1)
}

// GetApp mocks the GetApp method
func (m *MockAppService) GetApp(organizationID, appID string) (models.App, error) {
	args := m.Called(organizationID, appID)
	return args.Get(0).(models.App), args.Error(1)
}

// DeleteApp mocks the DeleteApp method
func (m *MockAppService) DeleteApp(organizationID, appID string) error {
	args := m.Called(organizationID, appID)
	return args.Error(0)
}
