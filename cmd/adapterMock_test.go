package cmd_test

import (
	"bytes"
	"errors"
	"log"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/Seicrypto/torcontroller/internal/singleton/logger"
)

// MockSocket is a mock implementation of ConnectionAdapter.
type MockSocket struct {
	ResponseMap map[string]string // Maps written commands to responses
	CloseSignal chan struct{}     // Signal to stop the goroutine
	Error       error             // Simulated connection error
}

// Dial simulates establishing a connection and returns a mock net.Conn.
func (m *MockSocket) Dial() (net.Conn, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	client, server := net.Pipe() // Simulate a connection
	go func() {
		defer server.Close()

		buf := make([]byte, 1024)
		for {
			select {
			case <-m.CloseSignal:
				return // Exit the loop when the close signal is received
			default:
				n, err := server.Read(buf)
				if err != nil {
					return // Connection closed
				}

				if response, exists := m.ResponseMap[string(buf[:n])]; exists {
					server.Write([]byte(response)) // Write the corresponding response
				} else {
					server.Write([]byte("5XX Command not recognized\n"))
				}
			}
		}
	}()

	return client, nil
}

// MockConnectionAdapter emulates ConnectionAdapter.
type MockConnectionAdapter struct {
	MockConn *MockSocket
	FailDial bool
}

// MockFileInfo is a mock implementation of os.FileInfo for testing.
type MockFileInfo struct {
	content []byte
	mode    os.FileMode
	uid     uint32
	gid     uint32
}

func (m *MockFileInfo) Name() string       { return "mockfile" }
func (m *MockFileInfo) Size() int64        { return 0 }
func (m *MockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool        { return false }
func (m *MockFileInfo) Sys() interface{} {
	return &syscall.Stat_t{
		Uid: m.uid,
		Gid: m.gid,
	}
}

// MockFileSystem is a mock implementation of FileSystem for testing.
type MockFileSystem struct {
	Files         map[string]*MockFileInfo
	ProcessExists map[int]bool
	Error         error
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	info, exists := m.Files[name]
	if !exists {
		return nil, errors.New("file not found")
	}
	return info, nil
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	info, exists := m.Files[name]
	if !exists {
		return nil, errors.New("file not found")
	}
	return info.content, nil
}

// FindProcess simulates finding a process by PID
func (fs *MockFileSystem) FindProcess(pid int) (*os.Process, error) {
	if exists := fs.ProcessExists[pid]; exists {
		return &os.Process{Pid: pid}, nil
	}
	return nil, errors.New("process not found")
}

// Remove simulates removing a file
func (fs *MockFileSystem) Remove(filename string) error {
	if _, exists := fs.Files[filename]; exists {
		delete(fs.Files, filename)
		return nil
	}
	return errors.New("file not found")
}

type MockLogger struct {
	Logs []string
}

func NewMockLogger() *logger.Logger {
	return &logger.Logger{
		Logger: log.New(&bytes.Buffer{}, "MOCK: ", log.LstdFlags),
	}
}
