package initializer_test

// func TestCheckTorService(t *testing.T) {
// 	mockRunner := &MockCommandRunner{
// 		CommandResponses: map[string]string{
// 			`sudo systemctl show tor`: "LoadState=loaded",
// 			`.*`:                      "Unexpected response",
// 		},
// 		CommandErrors: nil,
// 	}

// 	initializer := initializer.NewInitializer(&MockTemplates{}, mockRunner, &MockFileSystem{})

// 	if !initializer.CheckTorService() {
// 		t.Errorf("expected CheckTorService to return true for valid service")
// 	}

// 	// Test for an invalid response
// 	mockRunner.CommandResponses["sudo systemctl show tor"] = "LoadState=failed"
// 	if initializer.CheckTorService() {
// 		t.Errorf("expected CheckTorService to return false for invalid service")
// 	}
// }

// func TestCheckPrivoxyService(t *testing.T) {
// 	mockRunner := &MockCommandRunner{
// 		CommandResponses: map[string]string{
// 			`sudo systemctl show privoxy`: "LoadState=loaded",
// 			`.*`:                          "Unexpected response",
// 		},
// 		CommandErrors: nil,
// 	}

// 	initializer := initializer.NewInitializer(&MockTemplates{}, mockRunner, &MockFileSystem{})

// 	if !initializer.CheckPrivoxyService() {
// 		t.Errorf("expected CheckPrivoxyService to return true for valid service")
// 	}

// 	// Test for an invalid response
// 	mockRunner.CommandResponses["sudo systemctl show privoxy"] = "LoadState=inactive"
// 	if initializer.CheckPrivoxyService() {
// 		t.Errorf("expected CheckPrivoxyService to return false for invalid service")
// 	}
// }

// func TestCheckServiceFile(t *testing.T) {
// 	tests := []struct {
// 		serviceName   string
// 		mockResponses map[string]string
// 		mockErrors    map[string]error
// 		expectedValid bool
// 	}{
// 		{
// 			serviceName: "tor",
// 			mockResponses: map[string]string{
// 				`sudo systemctl show tor`: "LoadState=loaded",
// 				`.*`:                      "Unexpected response",
// 			},
// 			mockErrors:    nil,
// 			expectedValid: true,
// 		},
// 		{
// 			serviceName: "privoxy",
// 			mockResponses: map[string]string{
// 				"sudo systemctl show privoxy": "LoadState=failed",
// 			},
// 			mockErrors:    nil,
// 			expectedValid: false,
// 		},
// 		{
// 			serviceName: "tor",
// 			mockResponses: map[string]string{
// 				"sudo systemctl show tor": "",
// 			},
// 			mockErrors: map[string]error{
// 				"sudo systemctl show tor": errors.New("command failed"),
// 			},
// 			expectedValid: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.serviceName, func(t *testing.T) {
// 			mockRunner := &MockCommandRunner{
// 				CommandResponses: tt.mockResponses,
// 				CommandErrors:    tt.mockErrors,
// 			}

// 			init := initializer.NewInitializer(&MockTemplates{}, mockRunner, &MockFileSystem{})
// 			valid := init.CheckServiceFile(tt.serviceName)

// 			if valid != tt.expectedValid {
// 				t.Errorf("expected validity to be %v, got %v for service %s", tt.expectedValid, valid, tt.serviceName)
// 			}
// 		})
// 	}
// }
