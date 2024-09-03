package members

import (
	"github.com/stretchr/testify/mock"
)

// MockMemberService is a mock implementation of the MemberServicer interface
type MockMemberService struct {
	mock.Mock
}

// ListMembers mocks the ListMembers method
func (m *MockMemberService) ListMembers(orgID string) ([]Member, error) {
	args := m.Called(orgID)
	return args.Get(0).([]Member), args.Error(1)
}

// CreateMemberForOrg mocks the CreateMemberForOrg method
func (m *MockMemberService) CreateMemberForOrg(orgID string, member Member) (Member, error) {
	args := m.Called(orgID, member)
	return args.Get(0).(Member), args.Error(1)
}

// DeleteMember mocks the DeleteMember method
func (m *MockMemberService) DeleteMember(orgID, memberID string) error {
	args := m.Called(orgID, memberID)
	return args.Error(0)
}

// NewMockMemberService creates a new instance of MockMemberService
func NewMockMemberService() *MockMemberService {
	return &MockMemberService{}
}
