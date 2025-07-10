package env

import (
	"github.com/Hyphen/cli/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockEnvService is a mock implementation of the EnvServicer interface
type MockEnvService struct {
	mock.Mock
}

// GetEnvironment mocks the GetEnvironment method
func (m *MockEnvService) GetEnvironment(organizationId, appId, env string) (models.Environment, bool, error) {
	args := m.Called(organizationId, appId, env)
	return args.Get(0).(models.Environment), args.Bool(1), args.Error(2)
}

// PutEnv mocks the PutEnv method
func (m *MockEnvService) PutEnv(organizationId, appId, env string) error {
	args := m.Called(organizationId, appId, env)
	return args.Error(0)
}

// GetEnv mocks the GetEnv method
func (m *MockEnvService) GetEnv(organizationId, appId, env string) (models.Env, error) {
	args := m.Called(organizationId, appId, env)
	return args.Get(0).(models.Env), args.Error(1)
}

// ListEnvs mocks the ListEnvs method
func (m *MockEnvService) ListEnvs(organizationId, appId string, size, page int) ([]models.Env, error) {
	args := m.Called(organizationId, appId, size, page)
	return args.Get(0).([]models.Env), args.Error(1)
}

// NewMockEnvService creates a new instance of MockEnvService
func NewMockEnvService() *MockEnvService {
	return &MockEnvService{}
}
