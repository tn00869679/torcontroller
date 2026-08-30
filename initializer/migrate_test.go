package initializer_test

import (
	"strings"
	"testing"

	"github.com/tn00869679/torcontroller/initializer"
)

// A torrc as earlier versions shipped it: control port and cookie auth, the
// hardcoded password hash, and none of the transparent proxy settings.
const legacyTorrc = `ControlPort 9051
HashedControlPassword 16:7F7AC11D9E823EE460F1630CF5F31D1C45E733DE248867000A347C25AB

CookieAuthentication 1
CookieAuthFile /var/lib/tor/control.authcookie
`

const legacyUnit = `[Unit]
Description=Anonymizing overlay network for TCP
[Service]
User=root
ExecStart=/usr/bin/tor --runasdaemon 0 --defaults-torrc /usr/share/tor/defaults-torrc -f /etc/tor/torrc
`

// recordingRunner captures what the migration asked to run, so tests can
// assert that files the operator owns were left alone.
type recordingRunner struct {
	calls []string
	fail  map[string]error
}

func (r *recordingRunner) Run(name string, args ...string) (string, error) {
	command := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, command)
	for pattern, err := range r.fail {
		if strings.Contains(command, pattern) {
			return "", err
		}
	}
	return "", nil
}

func (r *recordingRunner) ran(substring string) bool {
	for _, call := range r.calls {
		if strings.Contains(call, substring) {
			return true
		}
	}
	return false
}

// writtenTorrc returns the contents the migration staged for installation, or
// an empty string if it decided nothing needed changing.
func writtenTorrc(t *testing.T, runner *recordingRunner, fs *MockFileSystem) string {
	t.Helper()
	staged, exists := fs.Files[initializer.MigrateStagingPath]
	if !exists {
		return ""
	}
	if !runner.ran("mv " + initializer.MigrateStagingPath + " " + initializer.MigrateTorrcPath) {
		t.Error("torrc was staged but never moved into place")
	}
	return string(staged.content)
}

func migrateWith(torrc string, unit *string) (*recordingRunner, *MockFileSystem, *initializer.Initializer) {
	files := map[string]*MockFileInfo{
		initializer.MigrateTorrcPath: {content: []byte(torrc), exists: true},
	}
	if unit != nil {
		files[initializer.LegacyUnitPath] = &MockFileInfo{content: []byte(*unit), exists: true}
	}
	fs := &MockFileSystem{Files: files}
	runner := &recordingRunner{}
	return runner, fs, initializer.NewInitializer(&MockTemplates{}, runner, fs)
}

func TestMigrationAddsEveryDirectiveTheRulesDependOn(t *testing.T) {
	runner, fs, init := migrateWith(legacyTorrc, nil)

	result, err := init.Migrate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, directive := range []string{"TransPort", "DNSPort", "AutomapHostsOnResolve",
		"AutomapHostsSuffixes", "VirtualAddrNetworkIPv4", "VirtualAddrNetworkIPv6"} {
		found := false
		for _, added := range result.AddedDirectives {
			if added == directive {
				found = true
			}
		}
		if !found {
			t.Errorf("%s should have been added, got %v", directive, result.AddedDirectives)
		}
	}

	written := writtenTorrc(t, runner, fs)
	if !strings.Contains(written, "TransPort 127.0.0.1:9040") || !strings.Contains(written, "TransPort [::1]:9041") {
		t.Error("both TransPort lines are needed; the IPv6 rules target the second one")
	}
	// The operator's existing settings must survive.
	if !strings.Contains(written, "ControlPort 9051") || !strings.Contains(written, "CookieAuthFile /var/lib/tor/control.authcookie") {
		t.Error("existing directives were lost; the migration must edit, not replace")
	}
}

// The hash was identical on every installation, so anyone knowing its
// plaintext could drive the control port of any machine still carrying it.
func TestMigrationRemovesTheSharedControlPassword(t *testing.T) {
	runner, fs, init := migrateWith(legacyTorrc, nil)

	result, err := init.Migrate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.RemovedPassword {
		t.Fatal("the shipped password hash should have been removed")
	}
	if strings.Contains(writtenTorrc(t, runner, fs), "16:7F7AC11D9E") {
		t.Error("the shared hash is still in the file")
	}
}

// A password the operator chose is theirs. Only the one we shipped is known to
// be unsafe.
func TestMigrationKeepsAPasswordTheOperatorSet(t *testing.T) {
	custom := "ControlPort 9051\nHashedControlPassword 16:AAAABBBBCCCCDDDDEEEEFFFF0000111122223333444455556\n"
	runner, fs, init := migrateWith(custom, nil)

	result, err := init.Migrate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RemovedPassword {
		t.Error("only the shipped hash should be removed")
	}
	if !strings.Contains(writtenTorrc(t, runner, fs), "16:AAAABBBB") {
		t.Error("the operator's own password was removed")
	}
}

// Someone who narrowed the range to avoid clashing with their IPv6 LAN must
// not have that undone by an upgrade.
func TestMigrationDoesNotOverwriteSettingsAlreadyPresent(t *testing.T) {
	tuned := `ControlPort 9051
TransPort 127.0.0.1:9040
DNSPort 5353
AutomapHostsOnResolve 1
AutomapHostsSuffixes .
VirtualAddrNetworkIPv4 10.192.0.0/10
VirtualAddrNetworkIPv6 [fd99::]/16
`
	runner, fs, init := migrateWith(tuned, nil)

	result, err := init.Migrate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.AddedDirectives) != 0 {
		t.Errorf("nothing should have been added, got %v", result.AddedDirectives)
	}
	if !result.AlreadyUpToDate {
		t.Error("a current configuration should report itself up to date")
	}
	if written := writtenTorrc(t, runner, fs); written != "" {
		t.Errorf("torrc should not have been rewritten at all, but was:\n%s", written)
	}
}

// Tor ignores commented lines, so treating them as configured would leave the
// installation broken in exactly the way this migration exists to prevent.
func TestMigrationTreatsCommentedDirectivesAsAbsent(t *testing.T) {
	commented := "ControlPort 9051\n#TransPort 127.0.0.1:9040\n#DNSPort 5353\n"
	runner, fs, init := migrateWith(commented, nil)

	result, err := init.Migrate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.AddedDirectives) == 0 {
		t.Fatal("commented directives must not count as present")
	}
	written := writtenTorrc(t, runner, fs)
	if strings.Count(written, "TransPort 127.0.0.1:9040") != 2 {
		// One commented, one added.
		t.Errorf("expected the commented line kept and a live one added, got:\n%s", written)
	}
}

// Left in place the old unit keeps Tor running as root, where the uid
// exemption cannot match and the rules redirect Tor into itself.
func TestMigrationRetiresTheUnitWeShipped(t *testing.T) {
	unit := legacyUnit
	runner, _, init := migrateWith(legacyTorrc, &unit)

	result, err := init.Migrate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.MovedLegacyUnit {
		t.Fatal("the legacy unit should have been retired")
	}
	if !runner.ran("mv " + initializer.LegacyUnitPath + " " + initializer.LegacyUnitBackup) {
		t.Errorf("expected the unit to be moved aside, calls were: %v", runner.calls)
	}
	if !runner.ran("systemctl daemon-reload") {
		t.Error("systemd must be reloaded or the retired unit stays in effect")
	}
	if runner.ran("rm ") {
		t.Error("the unit must be moved, not deleted: an operator may have edited it")
	}
}

// A unit at that path which we did not write belongs to the operator.
func TestMigrationLeavesAUnitItDidNotShipAlone(t *testing.T) {
	foreign := "[Service]\nExecStart=/usr/bin/tor -f /etc/tor/torrc\n"
	runner, _, init := migrateWith(legacyTorrc, &foreign)

	result, err := init.Migrate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MovedLegacyUnit {
		t.Error("a unit without our marker must not be touched")
	}
	if runner.ran("mv " + initializer.LegacyUnitPath) {
		t.Error("the operator's unit was moved")
	}
}

// postinst runs this on every upgrade, so a second pass must be a no-op rather
// than appending the directives again.
func TestMigrationIsIdempotent(t *testing.T) {
	runner, fs, init := migrateWith(legacyTorrc, nil)

	if _, err := init.Migrate(); err != nil {
		t.Fatalf("first pass failed: %v", err)
	}
	migrated := writtenTorrc(t, runner, fs)
	if migrated == "" {
		t.Fatal("the first pass wrote nothing")
	}

	secondRunner, secondFS, secondInit := migrateWith(migrated, nil)
	result, err := secondInit.Migrate()
	if err != nil {
		t.Fatalf("second pass failed: %v", err)
	}
	if !result.AlreadyUpToDate {
		t.Errorf("the second pass changed something: added=%v password=%v",
			result.AddedDirectives, result.RemovedPassword)
	}
	if written := writtenTorrc(t, secondRunner, secondFS); written != "" {
		t.Error("the second pass rewrote torrc")
	}
}
