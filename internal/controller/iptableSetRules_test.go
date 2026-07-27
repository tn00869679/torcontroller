package controller_test

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/Seicrypto/torcontroller/internal/controller"
)

// TestApplyIptablesRulesFactory tests the ApplyIptablesRulesFactory method.
func TestApplyIptablesRulesFactory(t *testing.T) {
	// Mock logger
	var logBuffer bytes.Buffer
	mockLogger := log.New(&logBuffer, "TEST: ", log.LstdFlags)

	// Mock CommandRunner
	mockCommandRunner := &MockCommandRunner{}
	mockCommandRunner.On("sudo", []string{"iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8118"}, "", nil)
	mockCommandRunner.On("sudo", []string{"iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", "443", "-j", "REDIRECT", "--to-ports", "8118"}, "", nil)

	// Initialize the CommandHandler
	handler := &controller.CommandHandler{
		Logger:        mockLogger,
		CommandRunner: mockCommandRunner,
	}

	// Call the method with Ipv4RulesApply
	err := handler.ApplyIptablesRulesFactory(controller.Ipv4RulesApply)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate logs for expected outputs
	logOutput := logBuffer.String()
	for _, rule := range controller.Ipv4RulesApply {
		commandLog := strings.Join(append([]string{rule.Command}, rule.Args...), " ")
		if !strings.Contains(logOutput, commandLog) {
			t.Errorf("expected log to contain rule: %s", commandLog)
		}
	}

	// Ensure successful completion log is present
	if !strings.Contains(logOutput, "All iptables rules applied successfully.") {
		t.Error("expected log to confirm all rules were applied successfully")
	}
}

func TestClearIptablesRulesFactory(t *testing.T) {
	// Mock logger
	mockLogger := NewMockLogger()

	// Initialize the MockCommandRunner
	mockCommandRunner := &MockCommandRunner{}
	mockCommandRunner.On("sudo", []string{"iptables", "-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8118"}, "", nil)
	mockCommandRunner.On("sudo", []string{"iptables", "-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "443", "-j", "REDIRECT", "--to-ports", "8118"}, "", nil)

	// Create the CommandHandler
	handler := &controller.CommandHandler{
		Logger:        mockLogger,
		CommandRunner: mockCommandRunner,
	}

	// Run the ClearIptablesRulesFactory
	err := handler.ClearIptablesRulesFactory(controller.Ipv4RulesClear)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check logger for expected messages
	logOutput := mockLogger.Writer().(*bytes.Buffer).String()

	// Verify log contains expected messages
	if !strings.Contains(logOutput, "[INFO] Clearing rule: iptables -t nat -D OUTPUT -p tcp --dport 80 -j REDIRECT --to-ports 8118") {
		t.Errorf("Expected log message not found for clearing port 80 rule")
	}
	if !strings.Contains(logOutput, "[INFO] Clearing rule: iptables -t nat -D OUTPUT -p tcp --dport 443 -j REDIRECT --to-ports 8118") {
		t.Errorf("Expected log message not found for clearing port 443 rule")
	}
	if !strings.Contains(logOutput, "[INFO] All iptables rules cleared successfully.") {
		t.Errorf("Expected success log message not found")
	}
}
