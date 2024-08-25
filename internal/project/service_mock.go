package project

import (
	"github.com/stretchr/testify/mock"
)

// MockProjectService is a mock implementation of the ProjectServicer interface
type MockProjectService struct {
	mock.Mock
}

// GetListProjects mocks the GetListProjects method
func (m *MockProjectService) GetListProjects(organizationID string, pageSize, pageNum int) ([]Project, error) {
	args := m.Called(organizationID, pageSize, pageNum)
	return args.Get(0).([]Project), args.Error(1)
}

// CreateProject mocks the CreateProject method
func (m *MockProjectService) CreateProject(organizationID, alternateID, name string) (Project, error) {
	args := m.Called(organizationID, alternateID, name)
	return args.Get(0).(Project), args.Error(1)
}

// GetProject mocks the GetProject method
func (m *MockProjectService) GetProject(organizationID, projectID string) (Project, error) {
	args := m.Called(organizationID, projectID)
	return args.Get(0).(Project), args.Error(1)
}

// DeleteProject mocks the DeleteProject method
func (m *MockProjectService) DeleteProject(organizationID, projectID string) error {
	args := m.Called(organizationID, projectID)
	return args.Error(0)
}
