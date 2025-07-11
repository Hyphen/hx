package timeprovider

import (
	"time"

	"github.com/stretchr/testify/mock"
)

// MockTimeProvider is a mock implementation of the TimeProvider interface
type MockTimeProvider struct {
	mock.Mock
}

// Ensure MockTimeProvider implements TimeProvider
var _ TimeProvider = (*MockTimeProvider)(nil)

// Now mocks the Now method
func (m *MockTimeProvider) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// IsExpired mocks the IsExpired method
func (m *MockTimeProvider) IsExpired(expiryTime int64) bool {
	args := m.Called(expiryTime)
	return args.Bool(0)
}

// NewMockTimeProvider creates a new instance of MockTimeProvider
func NewMockTimeProvider() *MockTimeProvider {
	return &MockTimeProvider{}
}
