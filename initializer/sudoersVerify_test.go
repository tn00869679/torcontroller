package initializer_test

import (
	"errors"
	"os"
	"testing"

	"github.com/Seicrypto/torcontroller/initializer"
)

func TestSudoersFileVerify(t *testing.T) {
	tests := []struct {
		name          string
		fileExists    bool
		fileMode      os.FileMode
		uid           uint32
		gid           uint32
		mockCommand   string
		mockOutput    string
		mockError     error
		expectedValid bool
	}{
		{
			name:          "Valid Sudoers File",
			fileExists:    true,
			fileMode:      0o440,
			uid:           0,
			gid:           0,
			mockCommand:   "sudo visudo -cf /etc/sudoers.d/torcontroller",
			mockOutput:    "Command executed successfully.",
			mockError:     nil,
			expectedValid: true,
		},
		{
			name:          "Sudoers File Missing",
			fileExists:    false,
			expectedValid: false,
		},
		{
			name:          "Invalid Permissions",
			fileExists:    true,
			fileMode:      0o644,
			uid:           0,
			gid:           0,
			expectedValid: false,
		},
		{
			name:          "Invalid Owner",
			fileExists:    true,
			fileMode:      0o440,
			uid:           1000,
			gid:           1000,
			expectedValid: false,
		},
		{
			name:          "Invalid Syntax",
			fileExists:    true,
			fileMode:      0o440,
			uid:           0,
			gid:           0,
			mockCommand:   "sudo visudo -cf /etc/sudoers.d/torcontroller",
			mockOutput:    "",
			mockError:     errors.New("syntax error"),
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFileSystem := &MockFileSystem{
				Files: map[string]*MockFileInfo{},
			}

			// Conditionally add the file to mockFileSystem
			if tt.fileExists {
				mockFileSystem.Files["/etc/sudoers.d/torcontroller"] = &MockFileInfo{
					mode:   tt.fileMode,
					uid:    tt.uid,
					gid:    tt.gid,
					exists: true, // Ensure the file is marked as existing
				}
			}

			mockRunner := &MockCommandRunner{
				CommandResponses: map[string]string{
					tt.mockCommand: tt.mockOutput,
				},
				CommandErrors: map[string]error{
					tt.mockCommand: tt.mockError,
				},
			}

			mockInitializer := initializer.NewInitializer(&MockTemplates{}, mockRunner, mockFileSystem)

			valid := mockInitializer.SudoersFileVerify()
			if valid != tt.expectedValid {
				t.Errorf("expected validity to be %v, got %v for %s", tt.expectedValid, valid, tt.name)
			}
		})
	}
}
