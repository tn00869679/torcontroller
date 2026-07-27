package controller

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/Seicrypto/torcontroller/internal/singleton/configuration"
	"github.com/Seicrypto/torcontroller/internal/singleton/logger"
)

type CommandHandler struct {
	Logger        *log.Logger
	Socket        ConnectionAdapter            // abstract socket interface
	CommandRunner CommandRunner                // CLI runner interface
	Config        *configuration.Configuration // Injected configuration
	FileSystem    FileSystem                   // Abstract file system
}

// ConnectionAdapter for abstract socket behavior
type ConnectionAdapter interface {
	Dial() (net.Conn, error)
}

// RealSocket is the actual socket adapter.
type RealSocket struct {
	Address string
}

func (r *RealSocket) Dial() (net.Conn, error) {
	return net.Dial("tcp", r.Address)
}

type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

type RealCommandRunner struct{}

func (r *RealCommandRunner) Run(name string, args ...string) (string, error) {
	var out, errBuf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		// Include both stdout and stderr in the error
		return out.String(), fmt.Errorf("%s: %w", errBuf.String(), err)
	}
	return out.String(), nil
}

// FileSystem interface for file-related operations
type FileSystem interface {
	ReadFile(filename string) ([]byte, error)
	FindProcess(pid int) (*os.Process, error)
	Remove(filename string) error
}

// RealFileSystem implements FileSystem using the actual OS
type RealFileSystem struct{}

func (fs *RealFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// FindProcess finds a process by its PID
func (fs *RealFileSystem) FindProcess(pid int) (*os.Process, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}
	return process, nil
}

// Remove deletes the file specified by the filename
func (fs *RealFileSystem) Remove(filename string) error {
	return os.Remove(filename)
}

func NewCommandHandler(
	socket ConnectionAdapter,
	runner CommandRunner,
	log *log.Logger,
	cfg *configuration.Configuration,
	fs FileSystem,
) *CommandHandler {
	if log == nil {
		log = logger.GetLogger().Logger
	}
	if cfg == nil {
		cfg = configuration.GetConfig() // Default to singleton if not provided
	}
	return &CommandHandler{
		Logger:        log,
		Socket:        socket,
		CommandRunner: runner,
		Config:        cfg,
		FileSystem:    fs,
	}
}
