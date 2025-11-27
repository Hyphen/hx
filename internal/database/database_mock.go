package database

import "github.com/stretchr/testify/mock"

type MockDatabase struct {
	mock.Mock
}

var _ Database = (*MockDatabase)(nil)

func (m *MockDatabase) GetSecret(key SecretKey) (Secret, bool) {
	args := m.Called(key)
	return args.Get(0).(Secret), args.Bool(1)
}

func (m *MockDatabase) UpsertSecret(key SecretKey, data string, version int) error {
	args := m.Called(key, data, version)
	return args.Error(0)
}
