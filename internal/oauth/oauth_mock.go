package oauth

import (
	"github.com/stretchr/testify/mock"
)

// MockOAuthService is a mock implementation of the OAuthServicer interface
type MockOAuthService struct {
	mock.Mock
}

// IsTokenExpired mocks the IsTokenExpired method
func (m *MockOAuthService) IsTokenExpired(expiryTime int64) bool {
	args := m.Called(expiryTime)
	return args.Bool(0)
}

// RefreshToken mocks the RefreshToken method
func (m *MockOAuthService) RefreshToken(refreshToken string) (*TokenResponse, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenResponse), args.Error(1)
}

// GetValidToken mocks the GetValidToken method
func (m *MockOAuthService) GetValidToken() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// StartOAuthServer mocks the StartOAuthServer method
func (m *MockOAuthService) StartOAuthServer() (*TokenResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenResponse), args.Error(1)
}
