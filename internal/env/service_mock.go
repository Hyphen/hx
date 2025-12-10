package env

import (
	"github.com/Hyphen/cli/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockEnvService is a mock implementation of the EnvServicer interface
type MockEnvService struct {
	mock.Mock
}

var _ EnvServicer = (*MockEnvService)(nil)

// GetEnvironment mocks the GetEnvironment method
func (m *MockEnvService) GetEnvironment(organizationId, projectId, environment string) (models.Environment, bool, error) {
	args := m.Called(organizationId, projectId, environment)
	return args.Get(0).(models.Environment), args.Bool(1), args.Error(2)
}

// PutEnvironmentEnv mocks the PutEnvironmentEnv method
func (m *MockEnvService) PutEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId int64, env models.Env) error {
	args := m.Called(organizationId, appId, environmentId, secretKeyId, env)
	return args.Error(0)
}

// GetEnvironmentEnv mocks the GetEnvironmentEnv method
func (m *MockEnvService) GetEnvironmentEnv(organizationId, appId, environmentId string, secretKeyId *int64, version *int) (models.Env, error) {
	args := m.Called(organizationId, appId, environmentId, secretKeyId, version)
	return args.Get(0).(models.Env), args.Error(1)
}

// ListEnvs mocks the ListEnvs method
func (m *MockEnvService) ListEnvs(organizationId, appId string, size, page int) ([]models.Env, error) {
	args := m.Called(organizationId, appId, size, page)
	return args.Get(0).([]models.Env), args.Error(1)
}

// ListEnvVersions mocks the ListEnvVersions method
func (m *MockEnvService) ListEnvVersions(organizationId, appId, environmentId string, size, page int) ([]models.Env, error) {
	args := m.Called(organizationId, appId, environmentId, size, page)
	return args.Get(0).([]models.Env), args.Error(1)
}

// ListEnvironments mocks the ListEnvironments method
func (m *MockEnvService) ListEnvironments(organizationId, projectId string, size, page int) ([]models.Environment, error) {
	args := m.Called(organizationId, projectId, size, page)
	return args.Get(0).([]models.Environment), args.Error(1)
}

// NewMockEnvService creates a new instance of MockEnvService
func NewMockEnvService() *MockEnvService {
	return &MockEnvService{}
}
