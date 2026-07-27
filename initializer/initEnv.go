package initializer

import (
	"fmt"
	"path/filepath"
)

func (i *Initializer) InitializeAllConfig() error {
	configs := map[string]string{
		"templates/privoxy.service":         "/etc/systemd/system/privoxy.service",
		"templates/tor.service":             "/etc/systemd/system/tor.service",
		"templates/privoxy/config":          "/etc/privoxy/config",
		"templates/tor/torrc":               "/etc/tor/torrc",
		"templates/sudoers.d/torcontroller": "/etc/sudoers.d/torcontroller",
		"templates/torcontroller.yml":       "/etc/torcontroller/torcontroller.yml",
	}

	// Override configuration files
	for src, dest := range configs {
		if err := i.WriteTemplateToFile(src, dest); err != nil {
			return fmt.Errorf("failed to write configuration for %s: %w", dest, err)
		}
	}

	// Setting file permissions
	if err := i.FileSystem.Chmod("/etc/sudoers.d/torcontroller", 0440); err != nil {
		return fmt.Errorf("failed to set permissions for /etc/sudoers.d/torcontroller: %w", err)
	}
	if err := i.FileSystem.Chmod("/etc/systemd/system/privoxy.service", 0644); err != nil {
		return fmt.Errorf("failed to set permissions for /etc/systemd/system/privoxy.service: %w", err)
	}
	if err := i.FileSystem.Chmod("/etc/systemd/system/tor.service", 0644); err != nil {
		return fmt.Errorf("failed to set permissions for /etc/systemd/system/tor.service: %w", err)
	}
	if err := i.FileSystem.Chmod("/etc/privoxy/config", 0644); err != nil {
		return fmt.Errorf("failed to set permissions for /etc/privoxy/config: %w", err)
	}
	if err := i.FileSystem.Chmod("/etc/torcontroller/torcontroller.yml", 0644); err != nil {
		return fmt.Errorf("failed to set permissions for /etc/torcontroller/torcontroller.yml: %w", err)
	}

	// Execute torcontroller newpassword
	if err := i.GenerateNewPassword(); err != nil {
		return fmt.Errorf("failed to generate new password: %w", err)
	}

	fmt.Println("[INFO] All configurations initialized successfully.")
	return nil
}

func (i *Initializer) WriteTemplateToFile(templatePath, destPath string) error {
	// Read the contents of the template
	content, err := i.Templates.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// Make sure the target directory exists
	destDir := filepath.Dir(destPath)
	if err := i.FileSystem.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	// Write the target file
	if err := i.FileSystem.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", destPath, err)
	}

	fmt.Printf("[INFO] Written %s to %s\n", templatePath, destPath)
	return nil
}

func (i *Initializer) GenerateNewPassword() error {
	cmd := []string{"torcontroller", "newpassword"}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("failed to execute torcontroller newpassword: %w", err)
	}

	fmt.Println("[INFO] New password generated successfully.")
	return nil
}
