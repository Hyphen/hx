package fsutil

import (
	"os"
)

// FileSystem interface defines the file operations we need
type FileSystem interface {
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Create(name string) (*os.File, error)
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	Remove(path string) error
}

// RealFileSystem implements FileSystem using actual OS calls
type RealFileSystem struct{}

func (rfs *RealFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (rfs *RealFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (rfs *RealFileSystem) Create(name string) (*os.File, error) {
	return os.Create(name)
}

func (rfs *RealFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (rfs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (rfs *RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

// NewFileSystem returns a new instance of RealFileSystem
func NewFileSystem() FileSystem {
	return &RealFileSystem{}
}
