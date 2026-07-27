package initializer_test

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type MockCommandRunner struct {
	CommandResponses map[string]string
	CommandErrors    map[string]error
}

func (m *MockCommandRunner) Run(name string, args ...string) (string, error) {
	command := fmt.Sprintf("%s %s", name, strings.Join(args, " "))

	// Check for exact matches
	if response, exists := m.CommandResponses[command]; exists {
		if err, hasError := m.CommandErrors[command]; hasError {
			return "", err
		}
		return response, nil
	}

	// Use canonical matching
	for pattern, response := range m.CommandResponses {
		matched, _ := regexp.MatchString(pattern, command)
		if matched {
			if err, hasError := m.CommandErrors[pattern]; hasError {
				return "", err
			}
			return response, nil
		}
	}

	return "", fmt.Errorf("no mock response defined for command: %s", command)
}

// MockTemplates is a mock implementation of TemplateProvider for testing.
type MockTemplates struct {
	Files map[string][]byte
	Error error
}

func (m *MockTemplates) ReadFile(name string) ([]byte, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	content, exists := m.Files[name]
	if !exists {
		return nil, errors.New("file not found")
	}
	return content, nil
}

// MockFileInfo is a mock implementation of os.FileInfo for testing.
type MockFileInfo struct {
	content []byte
	mode    os.FileMode
	uid     uint32
	gid     uint32
	exists  bool
	isDir   bool
}

func (m *MockFileInfo) Name() string       { return "mockfile" }
func (m *MockFileInfo) Size() int64        { return 0 }
func (m *MockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool {
	return m.isDir
}
func (m *MockFileInfo) Sys() interface{} {
	return &syscall.Stat_t{
		Uid: m.uid,
		Gid: m.gid,
	}
}

// MockFileSystem is a mock implementation of FileSystem for testing.
type MockFileSystem struct {
	Files       map[string]*MockFileInfo
	Error       error
	MkdirErrors map[string]error
	ChmodErrors map[string]error
	WriteErrors map[string]error
}

func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	if file, exists := m.Files[path]; exists && file.exists {
		return file, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	info, exists := m.Files[name]
	if !exists {
		return nil, errors.New("file not found")
	}
	return info.content, nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err, exists := m.MkdirErrors[path]; exists {
		return err
	}
	// Simulate creating a directory
	m.Files[path] = &MockFileInfo{mode: perm, exists: true}
	return nil
}

// Chmod mocks setting permissions for a file.
func (m *MockFileSystem) Chmod(name string, mode os.FileMode) error {
	if err, exists := m.ChmodErrors[name]; exists {
		return err
	}
	file, exists := m.Files[name]
	if !exists {
		return errors.New("file not found")
	}
	file.mode = mode
	return nil
}

// WriteFile mocks writing a file with content and permissions.
func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err, exists := m.WriteErrors[filename]; exists {
		return err
	}
	m.Files[filename] = &MockFileInfo{
		content: data,
		mode:    perm,
		exists:  true,
	}
	return nil
}

func (m *MockFileSystem) IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
