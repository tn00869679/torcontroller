package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"math/rand"

	"github.com/spf13/cobra"
)

const torrcPath = "/etc/tor/torrc"

// NewPasswordCmd creates a new hashed password and updates torrc.
var NewPasswordCmd = &cobra.Command{
	Use:   "newpassword [password]",
	Short: "Generate and set a new control password for Tor",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var password string
		if len(args) > 0 {
			password = args[0]
		} else {
			// Generate a random password
			password = generateRandomPassword(12)
			fmt.Printf("Generated random password: %s\n", password)
		}

		// Hash the password
		hashedPassword, err := hashPassword(password)
		if err != nil {
			fmt.Printf("Error hashing password: %v\n", err)
			return
		}

		// Update torrc
		err = updateTorrc(hashedPassword)
		if err != nil {
			fmt.Printf("Error updating torrc: %v\n", err)
			return
		}

		// Writing torrc changes nothing until Tor re-reads it. Without this the
		// command reported success while the running daemon still accepted only
		// the old password, and the new one worked no earlier than the next
		// reboot.
		if err := reloadTor(); err != nil {
			fmt.Printf("Password written to %s but Tor could not be reloaded: %v\n", torrcPath, err)
			fmt.Println("Run 'sudo systemctl reload tor' to apply it.")
			return
		}

		fmt.Println("New password successfully set and applied.")
		fmt.Println("Note: torcontroller itself authenticates with the control cookie.")
		fmt.Println("This password is only for other clients of Tor's control port.")
	},
}

// reloadTor makes the running daemon re-read torrc. Debian's unit maps reload
// to SIGHUP, which applies the new HashedControlPassword without dropping
// established circuits the way a restart would.
func reloadTor() error {
	var errBuf bytes.Buffer
	cmd := exec.Command("sudo", "systemctl", "reload", "tor")
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(errBuf.String()), err)
	}
	return nil
}

// hashPassword generates a hashed password using `tor --hash-password`.
func hashPassword(password string) (string, error) {
	var out, errBuf bytes.Buffer

	// Execute the `tor --hash-password` command
	cmd := exec.Command("tor", "--hash-password", password)
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %v, details: %s", err, errBuf.String())
	}

	// Extract the hashed password from the output
	outputLines := strings.Split(out.String(), "\n")
	for _, line := range outputLines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "16:") {
			return line, nil
		}
	}

	return "", fmt.Errorf("hashed password not found in output")
}

// updateTorrc updates the torrc file with the new hashed password.
func updateTorrc(hashedPassword string) error {
	file, err := os.OpenFile(torrcPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open torrc: %v", err)
	}
	defer file.Close()

	var content bytes.Buffer
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "HashedControlPassword") {
			content.WriteString(line + "\n")
		}
	}

	if scanner.Err() != nil {
		return fmt.Errorf("failed to read torrc: %v", scanner.Err())
	}

	content.WriteString(fmt.Sprintf("HashedControlPassword %s\n", strings.TrimSpace(hashedPassword)))

	// Write back to torrc
	err = os.WriteFile(torrcPath, content.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to torrc: %v", err)
	}

	return nil
}

// generateRandomPassword creates a random alphanumeric password.
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	password := make([]byte, length)
	for i := range password {
		password[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(password)
}
