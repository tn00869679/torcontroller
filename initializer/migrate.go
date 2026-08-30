package initializer

import (
	"fmt"
	"os"
	"strings"
)

// Paths the migration touches.
const (
	MigrateTorrcPath  = "/etc/tor/torrc"
	LegacyUnitPath    = "/etc/systemd/system/tor.service"
	LegacyUnitBackup  = "/etc/systemd/system/tor.service.torcontroller-backup"
	legacyUnitMarker  = "/usr/share/tor/defaults-torrc"
	legacyPasswordSum = "16:7F7AC11D9E823EE460F1630CF5F31D1C45E733DE248867000A347C25AB"
)

// requiredTorrcDirectives are the settings the redirection rules depend on.
// Absent any of them, `start` refuses rather than installing rules that point
// at ports nothing listens on -- so an incomplete migration costs the feature,
// not the network.
var requiredTorrcDirectives = []struct {
	Directive string
	Lines     []string
}{
	{"TransPort", []string{"TransPort 127.0.0.1:9040", "TransPort [::1]:9041"}},
	{"DNSPort", []string{"DNSPort 5353"}},
	{"AutomapHostsOnResolve", []string{"AutomapHostsOnResolve 1"}},
	{"AutomapHostsSuffixes", []string{"AutomapHostsSuffixes ."}},
	{"VirtualAddrNetworkIPv4", []string{"VirtualAddrNetworkIPv4 10.192.0.0/10"}},
	{"VirtualAddrNetworkIPv6", []string{"VirtualAddrNetworkIPv6 [fc00::]/7"}},
}

// MigrationResult records what changed, so the caller can tell the operator
// what happened to files they own.
type MigrationResult struct {
	AddedDirectives  []string
	RemovedPassword  bool
	MovedLegacyUnit  bool
	AlreadyUpToDate  bool
}

// Migrate brings an installation made by an earlier version in line with what
// the transparent proxy needs.
//
// It edits rather than replaces. postinst deliberately never overwrites a
// torrc the operator may have customised, which is correct, but it means an
// upgrade otherwise ships a binary expecting TransPort against a torrc that
// has none. Only the missing pieces are appended; anything already present,
// including a range the operator narrowed themselves, is left alone.
func (i *Initializer) Migrate() (MigrationResult, error) {
	var result MigrationResult

	contents, err := i.FileSystem.ReadFile(MigrateTorrcPath)
	if err != nil {
		return result, fmt.Errorf("failed to read %s: %w", MigrateTorrcPath, err)
	}
	original := string(contents)
	updated := original

	present := presentDirectives(original)
	var additions []string
	for _, required := range requiredTorrcDirectives {
		if present[required.Directive] {
			continue
		}
		additions = append(additions, required.Lines...)
		result.AddedDirectives = append(result.AddedDirectives, required.Directive)
	}
	if len(additions) > 0 {
		updated = strings.TrimRight(updated, "\n") + "\n" +
			"\n############### torcontroller transparent proxy ####################\n" +
			"## Added on upgrade. The redirection rules point at these ports; without\n" +
			"## them `torcontroller start` refuses to install any rules.\n" +
			strings.Join(additions, "\n") + "\n"
	}

	// The hash shipped with earlier versions was identical on every machine,
	// so anyone knowing its plaintext could drive the control port here. A
	// password the operator set themselves is left untouched.
	if strings.Contains(updated, legacyPasswordSum) {
		updated = removeLegacyPasswordLine(updated)
		result.RemovedPassword = true
	}

	if updated != original {
		if err := i.writeTorrc(updated); err != nil {
			return result, err
		}
	}

	moved, err := i.moveLegacyUnitAside()
	if err != nil {
		return result, err
	}
	result.MovedLegacyUnit = moved

	result.AlreadyUpToDate = len(result.AddedDirectives) == 0 && !result.RemovedPassword && !moved
	return result, nil
}

// presentDirectives reports which directives are in effect. Commented lines do
// not count: Tor ignores them, so treating them as configured would leave the
// installation broken in exactly the way this is meant to prevent.
func presentDirectives(contents string) map[string]bool {
	found := make(map[string]bool)
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if fields := strings.Fields(line); len(fields) > 0 {
			found[fields[0]] = true
		}
	}
	return found
}

func removeLegacyPasswordLine(contents string) string {
	var kept []string
	for _, line := range strings.Split(contents, "\n") {
		if strings.Contains(line, legacyPasswordSum) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// moveLegacyUnitAside retires the tor.service earlier versions installed.
//
// That unit shadowed Debian's, passed a --defaults-torrc path the tor package
// does not ship, and forced User=root. Left in place it keeps Tor running as
// root, where the uid exemption cannot match and the redirection rules send
// Tor into itself. It is moved rather than deleted: an operator who edited it
// would otherwise lose their changes with no way back.
func (i *Initializer) moveLegacyUnitAside() (bool, error) {
	contents, err := i.FileSystem.ReadFile(LegacyUnitPath)
	if err != nil {
		if i.FileSystem.IsNotExist(err) || os.IsNotExist(err) {
			return false, nil
		}
		// Unreadable for another reason: leave it be rather than guess.
		return false, nil
	}

	// Only retire the file we shipped. A unit written by the operator that
	// happens to sit at this path is theirs to keep.
	if !strings.Contains(string(contents), legacyUnitMarker) {
		return false, nil
	}

	if _, err := i.CommandRunner.Run("sudo", "mv", LegacyUnitPath, LegacyUnitBackup); err != nil {
		return false, fmt.Errorf("failed to move %s aside: %w", LegacyUnitPath, err)
	}
	if _, err := i.CommandRunner.Run("sudo", "systemctl", "daemon-reload"); err != nil {
		return true, fmt.Errorf("moved %s to %s but systemd was not reloaded: %w", LegacyUnitPath, LegacyUnitBackup, err)
	}
	return true, nil
}

// MigrateStagingPath is where the rewritten torrc is assembled before being
// moved into place. Staging through the FileSystem interface rather than
// os.CreateTemp keeps the step observable in tests, which matters here: the
// assertion that counts is what the file ends up containing.
const MigrateStagingPath = "/tmp/torrc-migrate.tmp"

func (i *Initializer) writeTorrc(contents string) error {
	if err := i.FileSystem.WriteFile(MigrateStagingPath, []byte(contents), 0644); err != nil {
		return fmt.Errorf("failed to stage the migrated torrc: %w", err)
	}
	if _, err := i.CommandRunner.Run("sudo", "mv", MigrateStagingPath, MigrateTorrcPath); err != nil {
		return fmt.Errorf("failed to install migrated torrc: %w", err)
	}
	if _, err := i.CommandRunner.Run("sudo", "chmod", "644", MigrateTorrcPath); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", MigrateTorrcPath, err)
	}
	return nil
}
