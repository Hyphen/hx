package user

import (
	"github.com/Hyphen/cli/pkg/errors"
)

// MockUserService is a mock implementation of UserServicer
type MockUserService struct {
	GetExecutionContextrmationFunc func() (ExecutionContext, error)
}

// Ensure MockUserService implements UserServicer
var _ UserServicer = (*MockUserService)(nil)

// GetExecutionContextrmation calls the mocked GetExecutionContextrmationFunc
func (m *MockUserService) GetExecutionContext() (ExecutionContext, error) {
	if m.GetExecutionContextrmationFunc != nil {
		return m.GetExecutionContextrmationFunc()
	}
	return ExecutionContext{}, errors.New("GetExecutionContextrmation: not implemented")
}

// NewMockUserService creates a new instance of MockUserService with default behavior
func NewMockUserService() *MockUserService {
	return &MockUserService{
		GetExecutionContextrmationFunc: func() (ExecutionContext, error) {
			return ExecutionContext{
				Member: Member{
					ID:   "mock-membership-id",
					Name: "Mock User",
					Organization: Organization{
						ID:   "mock-org-id",
						Name: "Mock Organization",
					},
					Rules: []Rule{},
				},
			}, nil
		},
	}
}
