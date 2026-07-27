package initializer

import (
	"fmt"
	"os"
	"path/filepath"
)

// PlaceTorServiceFile places the Tor service systemd file.
func (i *Initializer) PlaceTorServiceFile() error {
	content, err := i.Templates.ReadFile("templates/tor.service")
	if err != nil {
		return fmt.Errorf("failed to read tor service template: %w", err)
	}
	return i.writeServiceFile("/etc/systemd/system/tor.service", content)
}

// PlacePrivoxyServiceFile places the Privoxy service systemd file.
func (i *Initializer) PlacePrivoxyServiceFile() error {
	content, err := i.Templates.ReadFile("templates/privoxy.service")
	if err != nil {
		return fmt.Errorf("failed to read privoxy service template: %w", err)
	}
	return i.writeServiceFile("/etc/systemd/system/privoxy.service", content)
}

// writeServiceFile writes a service file to the specified path.
func (i *Initializer) writeServiceFile(path string, content []byte) error {
	tmpFile := "/tmp/service.tmp"
	if err := i.FileSystem.WriteFile(tmpFile, content, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	cmd := []string{"sudo", "mv", tmpFile, path}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("failed to move service file: %w", err)
	}
	return nil
}

// PlaceSudoersFile places the sudoers configuration file for torcontroller.
func (i *Initializer) PlaceSudoersFile() error {
	sudoersPath := "/etc/sudoers.d/torcontroller"

	content, err := i.Templates.ReadFile("templates/sudoers.d/torcontroller")
	if err != nil {
		return fmt.Errorf("failed to read sudoers template: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "torcontroller-sudoers-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary sudoers file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("failed to write to temporary sudoers file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary sudoers file: %w", err)
	}

	if _, err := i.CommandRunner.Run("sudo", "mv", tmpFile.Name(), sudoersPath); err != nil {
		return fmt.Errorf("failed to move sudoers file: %w", err)
	}
	if _, err := i.CommandRunner.Run("sudo", "chmod", "440", sudoersPath); err != nil {
		return fmt.Errorf("failed to set permissions on sudoers file: %w", err)
	}
	if _, err := i.CommandRunner.Run("sudo", "chown", "root:root", sudoersPath); err != nil {
		return fmt.Errorf("failed to set ownership on sudoers file: %w", err)
	}

	return nil
}

// PlaceTorrcConfig places the torrc configuration file at the expected location.
func (i *Initializer) PlaceTorrcConfig() error {
	const torrcPath = "/etc/tor/torrc"

	templateContent, err := i.Templates.ReadFile("templates/tor/torrc")
	if err != nil {
		return fmt.Errorf("failed to read torrc template: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "torrc-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for torrc: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(templateContent); err != nil {
		return fmt.Errorf("failed to write torrc template to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file for torrc: %w", err)
	}

	cmd := []string{"sudo", "mv", tmpFile.Name(), torrcPath}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("failed to move torrc file: %w", err)
	}

	cmd = []string{"sudo", "chmod", "644", torrcPath}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("failed to set permissions for torrc file: %w", err)
	}

	return nil
}

// PlacePrivoxyConfig places the privoxy configuration file at the expected location.
func (i *Initializer) PlacePrivoxyConfig() error {
	const privoxyConfigPath = "/etc/privoxy/config"

	templateContent, err := i.Templates.ReadFile("templates/privoxy/config")
	if err != nil {
		return fmt.Errorf("failed to read privoxy config template: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "privoxy-config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for privoxy config: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(templateContent); err != nil {
		return fmt.Errorf("failed to write privoxy config template to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file for privoxy config: %w", err)
	}

	cmd := []string{"sudo", "mv", tmpFile.Name(), privoxyConfigPath}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("failed to move privoxy config file: %w", err)
	}

	cmd = []string{"sudo", "chmod", "644", privoxyConfigPath}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("failed to set permissions for privoxy config file: %w", err)
	}

	return nil
}

// PlaceTorcontrollerYamlFile places the Torcontroller configuration file in the specified location.
func (i *Initializer) PlaceTorcontrollerYamlFile(path string) error {
	// Ensure the parent directory exists
	dir := filepath.Dir(path)
	if _, err := i.FileSystem.Stat(dir); os.IsNotExist(err) {
		if err := i.FileSystem.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory %s: %w", dir, err)
		}
	}

	content, err := i.Templates.ReadFile("templates/torcontroller.yml")
	if err != nil {
		return fmt.Errorf("failed to read torcontroller.yml template: %w", err)
	}
	tmpFile, err := os.CreateTemp("", "torcontroller-yaml-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary configuration file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("failed to write configuration template to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary configuration file: %w", err)
	}

	cmd := []string{"sudo", "mv", tmpFile.Name(), path}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("failed to move configuration file: %w", err)
	}

	cmd = []string{"sudo", "chmod", "600", path}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("failed to set permissions for configuration file: %w", err)
	}

	fmt.Println("[INFO] Configuration file placed successfully.")
	return nil
}
