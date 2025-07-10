package user

import (
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/errors"
)

// MockUserService is a mock implementation of UserServicer
type MockUserService struct {
	GetExecutionContextrmationFunc func() (models.ExecutionContext, error)
}

// Ensure MockUserService implements UserServicer
var _ UserServicer = (*MockUserService)(nil)

// GetExecutionContextrmation calls the mocked GetExecutionContextrmationFunc
func (m *MockUserService) GetExecutionContext() (models.ExecutionContext, error) {
	if m.GetExecutionContextrmationFunc != nil {
		return m.GetExecutionContextrmationFunc()
	}
	return models.ExecutionContext{}, errors.New("GetExecutionContextrmation: not implemented")
}

// NewMockUserService creates a new instance of MockUserService with default behavior
func NewMockUserService() *MockUserService {
	return &MockUserService{
		GetExecutionContextrmationFunc: func() (models.ExecutionContext, error) {
			return models.ExecutionContext{
				Member: models.Member{
					ID:   "mock-membership-id",
					Name: "Mock User",
					Organization: models.OrganizationReference{
						ID:   "mock-org-id",
						Name: "Mock Organization",
					},
					Rules: []models.Rule{},
				},
			}, nil
		},
	}
}
