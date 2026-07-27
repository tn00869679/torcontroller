package controller_test

import (
	"testing"

	"github.com/Seicrypto/torcontroller/internal/controller"
)

func TestRealCommandRunner_Run(t *testing.T) {
	runner := &controller.RealCommandRunner{}

	t.Run("test simple command", func(t *testing.T) {
		output, err := runner.Run("echo", "hello")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		expected := "hello\n"
		if output != expected {
			t.Errorf("expected %q, got %q", expected, output)
		}
	})

	t.Run("test failing command", func(t *testing.T) {
		output, err := runner.Run("ls", "/nonexistent-directory")
		if err == nil {
			t.Fatalf("expected an error, got nil")
		}
		if len(output) > 0 {
			t.Errorf("expected no output, got %q", output)
		}
	})
}
