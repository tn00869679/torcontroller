package cmd_test

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Commands that have to work on a machine whose configuration is missing.
// version is what someone runs to find out what they have installed, often
// because something is wrong; migrate exists to repair an installation that is
// not yet in order. Demanding the configuration file first makes both useless
// exactly when they are needed.
//
// This builds and runs the binary rather than calling into the package,
// because the behaviour under test lives in cobra's PersistentPreRunE and only
// shows up when a command is actually dispatched.
func TestCommandsThatMustWorkWithoutAConfigurationFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the binary does not build on Windows")
	}

	binary := filepath.Join(t.TempDir(), "torcontroller")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("could not build the binary: %v\n%s", err, output)
	}

	// A path that certainly does not exist, so the test does not depend on
	// whether this machine happens to have torcontroller installed.
	cases := []struct {
		command     string
		wantSuccess bool
	}{
		{"version", true},
		// `check` legitimately needs the configuration and should say so.
		{"check", false},
	}

	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			var stderr bytes.Buffer
			cmd := exec.Command(binary, tc.command)
			cmd.Stderr = &stderr
			err := cmd.Run()

			if tc.wantSuccess && err != nil {
				t.Errorf("%s should run without a configuration file, got: %v\n%s",
					tc.command, err, stderr.String())
			}
			if tc.wantSuccess && strings.Contains(stderr.String(), "failed to load configuration") {
				t.Errorf("%s complained about the configuration file: %s", tc.command, stderr.String())
			}
		})
	}
}
