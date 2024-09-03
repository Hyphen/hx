package secretkey

import (
	"github.com/stretchr/testify/mock"
)

// MockSecretKey is a mock implementation of the SecretKeyer interface
type MockSecretKey struct {
	mock.Mock
}

// Base64 mocks the Base64 method
func (m *MockSecretKey) Base64() string {
	args := m.Called()
	return args.String(0)
}

// HashSHA mocks the HashSHA method
func (m *MockSecretKey) HashSHA() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// Encrypt mocks the Encrypt method
func (m *MockSecretKey) Encrypt(message string) (string, error) {
	args := m.Called(message)
	return args.String(0), args.Error(1)
}

// Decrypt mocks the Decrypt method
func (m *MockSecretKey) Decrypt(encryptedMessage string) (string, error) {
	args := m.Called(encryptedMessage)
	return args.String(0), args.Error(1)
}

// NewMockSecretKey creates a new instance of MockSecretKey
func NewMockSecretKey() *MockSecretKey {
	return &MockSecretKey{}
}
