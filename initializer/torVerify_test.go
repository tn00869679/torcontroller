package initializer_test

import (
	"errors"
	"testing"

	"github.com/Seicrypto/torcontroller/initializer"
)

func TestVerifyTorrcConfig(t *testing.T) {
	tests := []struct {
		name          string
		mockCommand   string
		mockOutput    string
		mockError     error
		expectedValid bool
	}{
		{
			name:          "ValidTorrcConfig",
			mockCommand:   "sudo tor --verify-config",
			mockOutput:    "Configuration valid",
			mockError:     nil,
			expectedValid: true,
		},
		{
			name:          "InvalidTorrcConfig",
			mockCommand:   "sudo tor --verify-config",
			mockOutput:    "",
			mockError:     errors.New("invalid configuration"),
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := &MockCommandRunner{
				CommandResponses: map[string]string{
					tt.mockCommand: tt.mockOutput,
				},
				CommandErrors: map[string]error{
					tt.mockCommand: tt.mockError,
				},
			}

			init := initializer.NewInitializer(&MockTemplates{}, mockRunner, &MockFileSystem{})
			valid := init.VerifyTorrcConfig()

			if valid != tt.expectedValid {
				t.Errorf("expected validity to be %v, got %v for %s", tt.expectedValid, valid, tt.name)
			}
		})
	}
}
