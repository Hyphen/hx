package zelda

import (
	"github.com/stretchr/testify/mock"
)

// MockZeldaService is a mock implementation of the ZeldaServicer interface
type MockZeldaService struct {
	mock.Mock
}

// CreateCode mocks the CreateCode method
func (m *MockZeldaService) CreateCode(organizationID string, code Code) (Code, error) {
	args := m.Called(organizationID, code)
	return args.Get(0).(Code), args.Error(1)
}

// CreateQRCode mocks the CreateQRCode method
func (m *MockZeldaService) CreateQRCode(organizationID, codeId, title string) (QR, error) {
	args := m.Called(organizationID, codeId, title)
	return args.Get(0).(QR), args.Error(1)
}

// ListDomains mocks the ListDomains method
func (m *MockZeldaService) ListDomains(organizationID string, pageSize, pageNum int) ([]DomainInfo, error) {
	args := m.Called(organizationID, pageSize, pageNum)
	return args.Get(0).([]DomainInfo), args.Error(1)
}

// NewMockZeldaService creates a new instance of MockZeldaService
func NewMockZeldaService() *MockZeldaService {
	return &MockZeldaService{}
}
