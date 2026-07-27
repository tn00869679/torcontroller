package initializer_test

import (
	"fmt"
	"testing"

	"github.com/Seicrypto/torcontroller/initializer"
	"github.com/stretchr/testify/assert"
)

// TestPlaceTorrcConfig tests the PlaceTorrcConfig method.
func TestPlaceTorrcConfig(t *testing.T) {
	tests := []struct {
		name          string
		templateError error
		templateFile  string
		mockCommands  map[string]string
		mockErrors    map[string]error
		expectedError string
	}{
		{
			name:         "Success",
			templateFile: "SocksPort 9050\nControlPort 9051\n",
			mockCommands: map[string]string{
				`sudo mv .* /etc/tor/torrc`:     "Success",
				`sudo chmod 644 /etc/tor/torrc`: "Success",
			},
			expectedError: "",
		},
		{
			name:          "Template Read Error",
			templateError: fmt.Errorf("template read error"),
			expectedError: "failed to read torrc template",
		},
		{
			name:         "Command Error During Move",
			templateFile: "SocksPort 9050\nControlPort 9051\n",
			mockCommands: map[string]string{
				`sudo mv .* /etc/tor/torrc`: "",
			},
			mockErrors: map[string]error{
				`sudo mv .* /etc/tor/torrc`: fmt.Errorf("move error"),
			},
			expectedError: "failed to move torrc file",
		},
		{
			name:         "Command Error During Chmod",
			templateFile: "SocksPort 9050\nControlPort 9051\n",
			mockCommands: map[string]string{
				`sudo mv .* /etc/tor/torrc`:     "Success",
				`sudo chmod 644 /etc/tor/torrc`: "",
			},
			mockErrors: map[string]error{
				`sudo chmod 644 /etc/tor/torrc`: fmt.Errorf("chmod error"),
			},
			expectedError: "failed to set permissions for torrc file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTemplates := &MockTemplates{
				Files: map[string][]byte{
					"templates/tor/torrc": []byte(tt.templateFile),
				},
				Error: tt.templateError,
			}

			mockRunner := &MockCommandRunner{
				CommandResponses: tt.mockCommands,
				CommandErrors:    tt.mockErrors,
			}

			mockFileSystem := &MockFileSystem{
				Files: make(map[string]*MockFileInfo),
			}

			init := initializer.NewInitializer(mockTemplates, mockRunner, mockFileSystem)

			err := init.PlaceTorrcConfig()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.expectedError)
			}
		})
	}
}

// TestPlacePrivoxyConfig tests the PlacePrivoxyConfig method.
func TestPlacePrivoxyConfig(t *testing.T) {
	tests := []struct {
		name          string
		templateError error
		templateFile  string
		mockCommands  map[string]string
		mockErrors    map[string]error
		expectedError string
	}{
		{
			name:         "Success",
			templateFile: "listen-address 127.0.0.1:8118\nforward-socks5 / 127.0.0.1:9050 .",
			mockCommands: map[string]string{
				`sudo mv .* /etc/privoxy/config`:     "Success",
				`sudo chmod 644 /etc/privoxy/config`: "Success",
			},
			expectedError: "",
		},
		{
			name:          "Template Read Error",
			templateError: fmt.Errorf("template read error"),
			expectedError: "failed to read privoxy config template",
		},
		{
			name:         "Command Error During Move",
			templateFile: "listen-address 127.0.0.1:8118\nforward-socks5 / 127.0.0.1:9050 .",
			mockCommands: map[string]string{
				`sudo mv .* /etc/privoxy/config`: "",
			},
			mockErrors: map[string]error{
				`sudo mv .* /etc/privoxy/config`: fmt.Errorf("move error"),
			},
			expectedError: "failed to move privoxy config file",
		},
		{
			name:         "Command Error During Chmod",
			templateFile: "listen-address 127.0.0.1:8118\nforward-socks5 / 127.0.0.1:9050 .",
			mockCommands: map[string]string{
				`sudo mv .* /etc/privoxy/config`:     "Success",
				`sudo chmod 644 /etc/privoxy/config`: "",
			},
			mockErrors: map[string]error{
				`sudo chmod 644 /etc/privoxy/config`: fmt.Errorf("chmod error"),
			},
			expectedError: "failed to set permissions for privoxy config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTemplates := &MockTemplates{
				Files: map[string][]byte{
					"templates/privoxy/config": []byte(tt.templateFile),
				},
				Error: tt.templateError,
			}

			mockRunner := &MockCommandRunner{
				CommandResponses: tt.mockCommands,
				CommandErrors:    tt.mockErrors,
			}

			mockFileSystem := &MockFileSystem{
				Files: make(map[string]*MockFileInfo),
			}

			init := initializer.NewInitializer(mockTemplates, mockRunner, mockFileSystem)

			err := init.PlacePrivoxyConfig()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.expectedError)
			}
		})
	}
}

// TestPlaceTorcontrollerYamlFile tests the functionality of PlaceTorcontrollerYamlFile.
func TestPlaceTorcontrollerYamlFile(t *testing.T) {
	tests := []struct {
		name          string
		dirExists     bool
		mkdirError    error
		templateError error
		templateFile  string
		mockCommands  map[string]string
		mockErrors    map[string]error
		expectedError string
	}{
		{
			name:         "Success",
			dirExists:    true,
			templateFile: "rate_limit:\n  min_read_rate: 20000\n  min_write_rate: 10000",
			mockCommands: map[string]string{
				`sudo mv .* /etc/torcontroller/torcontroller.yml`:     "Success",
				`sudo chmod 600 /etc/torcontroller/torcontroller.yml`: "Success",
			},
			expectedError: "",
		},
		{
			name:          "Directory Creation Error",
			dirExists:     false,
			mkdirError:    fmt.Errorf("mkdir error"),
			expectedError: "failed to create parent directory",
		},
		{
			name:          "Template Read Error",
			dirExists:     true,
			templateError: fmt.Errorf("template read error"),
			expectedError: "failed to read torcontroller.yml template",
		},
		{
			name:         "Command Error During Move",
			dirExists:    true,
			templateFile: "rate_limit:\n  min_read_rate: 20000\n  min_write_rate: 10000",
			mockCommands: map[string]string{
				`sudo mv .* /etc/torcontroller/torcontroller.yml`: "",
			},
			mockErrors: map[string]error{
				`sudo mv .* /etc/torcontroller/torcontroller.yml`: fmt.Errorf("move error"),
			},
			expectedError: "failed to move configuration file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFileSystem := &MockFileSystem{
				Files:       make(map[string]*MockFileInfo),
				MkdirErrors: map[string]error{},
			}
			if !tt.dirExists {
				mockFileSystem.MkdirErrors["/etc/torcontroller"] = tt.mkdirError
			}

			mockTemplates := &MockTemplates{
				Files: map[string][]byte{
					"templates/torcontroller.yml": []byte(tt.templateFile),
				},
				Error: tt.templateError,
			}

			mockRunner := &MockCommandRunner{
				CommandResponses: tt.mockCommands,
				CommandErrors:    tt.mockErrors,
			}

			init := initializer.NewInitializer(mockTemplates, mockRunner, mockFileSystem)
			err := init.PlaceTorcontrollerYamlFile("/etc/torcontroller/torcontroller.yml")

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.expectedError)
			}
		})
	}
}
