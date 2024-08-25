package fsutil

import (
	"os"
	"time"
)

// MockFileSystem is a mock implementation of FileSystem
type MockFileSystem struct {
	ReadFileFunc  func(filename string) ([]byte, error)
	WriteFileFunc func(filename string, data []byte, perm os.FileMode) error
	CreateFunc    func(name string) (*os.File, error)
	StatFunc      func(name string) (os.FileInfo, error)
	MkdirAllFunc  func(path string, perm os.FileMode) error
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	return m.ReadFileFunc(filename)
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return m.WriteFileFunc(filename, data, perm)
}

func (m *MockFileSystem) Create(name string) (*os.File, error) {
	return m.CreateFunc(name)
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	return m.StatFunc(name)
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return m.MkdirAllFunc(path, perm)
}

// MockFileInfo is a mock implementation of os.FileInfo
type MockFileInfo struct {
	NameFunc    func() string
	SizeFunc    func() int64
	ModeFunc    func() os.FileMode
	ModTimeFunc func() time.Time
	IsDirFunc   func() bool
	SysFunc     func() interface{}
}

func (m MockFileInfo) Name() string       { return m.NameFunc() }
func (m MockFileInfo) Size() int64        { return m.SizeFunc() }
func (m MockFileInfo) Mode() os.FileMode  { return m.ModeFunc() }
func (m MockFileInfo) ModTime() time.Time { return m.ModTimeFunc() }
func (m MockFileInfo) IsDir() bool        { return m.IsDirFunc() }
func (m MockFileInfo) Sys() interface{}   { return m.SysFunc() }
