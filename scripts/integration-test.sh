#!/bin/bash
#
# End-to-end test for the transparent proxy, run inside a privileged container:
#
#   docker run --rm --cap-add=NET_ADMIN --cap-add=NET_RAW \
#     -v "$PWD:/repo:ro" ghcr.io/tn00869679/torcontroller/torcontroller-test-env:dev \
#     bash /repo/scripts/integration-test.sh /path/to/torcontroller
#
# It reaches the real Tor network, so it needs outbound access and takes a
# minute or two to bootstrap.
#
# What it does NOT cover, so nobody mistakes a pass for full coverage:
#
#   * `systemctl start tor`. The systemctl replacement in the test image
#     reports `active (running)` with no process behind it, so the service
#     management path cannot be exercised here -- Tor is started directly
#     instead. That shim is also why the port preflight exists: a service
#     manager claiming success is not evidence that anything is listening.
#   * Debian packaging. Nothing here builds or installs the .deb.
#   * Behaviour on a host with real IPv6 connectivity. The IPv6 checks work
#     through Tor's virtual addresses, which are local, so they pass without
#     the machine having any IPv6 route of its own.
#
# Every "nothing happened" assertion is paired with a control that proves the
# measurement can see something when it is there. A check that cannot fail is
# worse than no check, so the script aborts if a control comes back empty.
#
# The checks were themselves verified by running this against binaries built
# with deliberate faults, to confirm each one fails for its own reason:
#
#   fault introduced                     check that caught it
#   -----------------------------------  ------------------------------------
#   DNS redirect rule removed            8 packets on the wire, matching the
#                                        control exactly, against 0 when whole
#   teardown skips the automap route     the route was left behind
#   migration ignores what is present    a second migration is not a no-op
#   port preflight always returns nil    start no longer refuses with Tor down
#
# The last one is worth reading closely: with the preflight gone, start applied
# the rules and only failed afterwards, when it tried to reach a control port
# that was not there. The network was already redirected at a Tor that did not
# exist by the time anything complained.

set -u

BINARY="${1:-/repo/torcontroller}"
CONFIG_DIR=/etc/torcontroller
CONFIG="$CONFIG_DIR/torcontroller.yml"
TORRC=/etc/tor/torrc
PASSED=0
FAILED=0

say()  { printf '\n=== %s ===\n' "$*"; }
ok()   { PASSED=$((PASSED + 1)); printf '  PASS  %s\n' "$*"; }
bad()  { FAILED=$((FAILED + 1)); printf '  FAIL  %s\n' "$*"; }
abort() { printf '\nABORT: %s\n' "$*" >&2; exit 2; }

# A response counts only if it is shaped like an address. An error page and an
# IP are both non-empty strings, and treating the first as success is how an
# earlier round of this work concluded that a broken path was working.
is_ipv4() { printf '%s' "$1" | grep -qE '^[0-9]{1,3}(\.[0-9]{1,3}){3}$'; }

fetch() { curl -s --max-time "${2:-35}" "$1" 2>/dev/null; }

# Count DNS queries leaving the machine. Only the external interface counts:
# redirected queries still travel over loopback, and counting those would
# report a leak that is not one.
dns_on_wire() {
    local iface capture
    iface=$(ip route show default | awk '{print $5}' | head -1)
    capture=$(mktemp)
    timeout 12 tcpdump -i "$iface" -n -c 10 'udp port 53' -w "$capture" >/dev/null 2>&1 &
    local pid=$!
    sleep 2
    getent hosts github.com  >/dev/null 2>&1
    getent hosts example.org >/dev/null 2>&1
    getent hosts wikipedia.org >/dev/null 2>&1
    sleep 3
    kill "$pid" 2>/dev/null
    wait "$pid" 2>/dev/null
    tcpdump -r "$capture" -n 2>/dev/null | grep -c . || true
}

wait_for_tor() {
    local i
    for i in $(seq 1 45); do
        if curl -s --max-time 8 --socks5-hostname 127.0.0.1:9050 \
             http://icanhazip.com >/dev/null 2>&1; then
            return 0
        fi
        sleep 3
    done
    return 1
}

# ---------------------------------------------------------------- preconditions
say "Preconditions"
[ "$(id -u)" -eq 0 ] || abort "must run as root"
[ -x "$BINARY" ]     || abort "torcontroller binary not found at $BINARY"
iptables -t nat -L >/dev/null 2>&1 || abort "no NET_ADMIN; run with --cap-add=NET_ADMIN"
for tool in curl tcpdump dig tor getent; do
    command -v "$tool" >/dev/null 2>&1 || abort "missing required tool: $tool"
done

# Prove the binary runs here before asserting anything about what it does. A
# binary built against a newer libc than this image carries fails at every
# invocation, which otherwise reads as a dozen broken features rather than one
# broken build.
if ! "$BINARY" version >/dev/null 2>&1; then
    abort "the binary does not run in this image: $("$BINARY" version 2>&1 | head -1)"
fi
ok "root, NET_ADMIN, required tools, and a binary that runs here"

mkdir -p "$CONFIG_DIR" /var/log/tor
cp /repo/initializer/templates/tor/torrc "$TORRC"
cp /repo/initializer/templates/torcontroller.yml "$CONFIG"

# --------------------------------------------------------------------- baseline
say "Baseline with no rules installed"
REAL_IP=$(fetch http://icanhazip.com 20)
is_ipv4 "$REAL_IP" || abort "no direct network access; the rest of the run would be meaningless"
ok "direct address is $REAL_IP"

BASELINE_DNS=$(dns_on_wire)
if [ "$BASELINE_DNS" -eq 0 ]; then
    abort "no DNS seen on the wire without rules; the capture is broken, so a later zero would prove nothing"
fi
ok "control: $BASELINE_DNS DNS packets observed on the wire before any rules"

# ------------------------------------------------------- preflight before tor
say "Preflight refuses while Tor is down"
if "$BINARY" start >/tmp/start-early.log 2>&1; then
    bad "start succeeded with Tor down; rules would point at closed ports"
else
    if grep -q "not listening" /tmp/start-early.log; then
        ok "start refused and named the missing listeners"
    else
        bad "start failed for some other reason: $(head -2 /tmp/start-early.log | tr '\n' ' ')"
    fi
fi
"$BINARY" stop >/dev/null 2>&1

# ------------------------------------------------------------------- start tor
say "Starting Tor directly"
install -m 02755 -o debian-tor -g debian-tor -d /run/tor
/usr/bin/tor --defaults-torrc /usr/share/tor/tor-service-defaults-torrc \
    -f "$TORRC" --RunAsDaemon 0 >/tmp/tor.log 2>&1 &
wait_for_tor || abort "Tor did not bootstrap; check outbound access"

TOR_IP=$(curl -s --max-time 20 --socks5-hostname 127.0.0.1:9050 http://icanhazip.com)
is_ipv4 "$TOR_IP" || abort "Tor is up but its SOCKS port returned no address"
ok "Tor is reachable, exit address $TOR_IP"

if [ "$(ps -o user= -p "$(pgrep -f 'tor --defaults-torrc' | head -1)" | tr -d ' ')" = "debian-tor" ]; then
    ok "Tor runs as debian-tor, which the uid exemption depends on"
else
    bad "Tor is not running as debian-tor; the exemption cannot match"
fi

# ------------------------------------------------------------ torrc mismatch
say "A torrc that disagrees with the configuration is refused"
cp "$CONFIG" /tmp/config-backup.yml
# Replace the file rather than appending: the shipped one already has a proxy
# block, and a second one is a YAML duplicate-key error rather than the
# mismatch this is meant to provoke.
cat > "$CONFIG" <<'MISMATCH'
rate_limit:
  min_read_rate: 10000
  min_write_rate: 5000
proxy:
  virtual_net_ipv4: 172.31.0.0/16
MISMATCH
if "$BINARY" start >/tmp/start-mismatch.log 2>&1; then
    bad "start accepted a range torrc does not hand out; traffic would leave Tor silently"
else
    if grep -q "disagree" /tmp/start-mismatch.log; then
        ok "start refused and explained the mismatch"
    else
        bad "start failed for some other reason: $(head -2 /tmp/start-mismatch.log | tr '\n' ' ')"
    fi
fi
"$BINARY" stop >/dev/null 2>&1
cp /tmp/config-backup.yml "$CONFIG"

# ----------------------------------------------------------------------- start
say "torcontroller start"
if "$BINARY" start >/tmp/start.log 2>&1 && grep -q "Done" /tmp/start.log; then
    ok "start reported success"
else
    bad "start failed: $(tail -3 /tmp/start.log | tr '\n' ' ')"
fi

RULE_COUNT=$(iptables -t nat -L TORCONTROLLER -n 2>/dev/null | tail -n +3 | grep -c .)
[ "$RULE_COUNT" -ge 8 ] && ok "chain holds $RULE_COUNT rules" || bad "chain holds only $RULE_COUNT rules"

# --------------------------------------------------------------------- traffic
say "Traffic goes through Tor"
for target in http://icanhazip.com https://icanhazip.com; do
    response=$(fetch "$target")
    if ! is_ipv4 "$response"; then
        bad "$target returned something that is not an address: $(printf '%.60s' "$response")"
    elif [ "$response" = "$REAL_IP" ]; then
        bad "$target returned the real address; traffic is not going through Tor"
    else
        ok "$target exits as $response"
    fi
done

for port in 8080 2222 12345; do
    code=$(curl -s --max-time 25 -o /dev/null -w '%{http_code}' "http://portquiz.net:$port" 2>/dev/null)
    [ "$code" = "200" ] && ok "port $port reaches the network" || bad "port $port returned '$code'"
done

V6=$(curl -6 -s --max-time 40 http://icanhazip.com 2>/dev/null)
if is_ipv4 "$V6" && [ "$V6" != "$REAL_IP" ]; then
    ok "an IPv6 request exits through Tor as $V6"
else
    bad "IPv6 request returned '$(printf '%.60s' "$V6")'"
fi

if ip -6 route show table local 2>/dev/null | grep -q 'fc00::/7'; then
    ok "the automap route is installed"
else
    bad "the automap route is missing; IPv6 redirection cannot work without it"
fi

# ------------------------------------------------------------------------- dns
say "DNS no longer leaves the machine"
LEAKED=$(dns_on_wire)
if [ "$LEAKED" -eq 0 ]; then
    ok "0 DNS packets on the wire, against $BASELINE_DNS in the control"
else
    bad "$LEAKED DNS packets still leaving the machine"
fi

# ------------------------------------------------------------------------ stop
say "torcontroller stop restores the host"
if "$BINARY" stop >/tmp/stop.log 2>&1; then
    ok "stop reported success"
else
    bad "stop failed: $(tail -3 /tmp/stop.log | tr '\n' ' ')"
fi

AFTER=$(fetch http://icanhazip.com 20)
[ "$AFTER" = "$REAL_IP" ] && ok "traffic returns to $REAL_IP" || bad "expected $REAL_IP after stop, got '$AFTER'"

iptables -t nat -L OUTPUT -n | tail -n +3 | grep -q TORCONTROLLER \
    && bad "the OUTPUT jump is still installed" || ok "no rules left in OUTPUT"
ip -6 route show table local 2>/dev/null | grep -q 'fc00::/7' \
    && bad "the automap route was left behind, which makes real ULA addresses unreachable" \
    || ok "the automap route was removed"

# -------------------------------------------------------------------- migration
say "Upgrading a configuration from an earlier version"
cat > "$TORRC" <<'LEGACY'
ControlPort 9051
HashedControlPassword 16:7F7AC11D9E823EE460F1630CF5F31D1C45E733DE248867000A347C25AB
CookieAuthentication 1
CookieAuthFile /var/lib/tor/control.authcookie
Log notice file /var/log/tor/notices.log
LEGACY
cat > /etc/systemd/system/tor.service <<'LEGACYUNIT'
[Service]
User=root
ExecStart=/usr/bin/tor --defaults-torrc /usr/share/tor/defaults-torrc -f /etc/tor/torrc
LEGACYUNIT

"$BINARY" migrate >/tmp/migrate.log 2>&1 || bad "migrate failed: $(tail -2 /tmp/migrate.log)"

[ "$(grep -c '^TransPort' "$TORRC")" -eq 2 ] \
    && ok "both TransPort lines were added" || bad "expected 2 TransPort lines"
grep -q '16:7F7AC11D9E' "$TORRC" \
    && bad "the shared control password survived" || ok "the shared control password was removed"
grep -q 'Log notice file' "$TORRC" \
    && ok "a directive the operator added survived" || bad "operator configuration was lost"
[ -f /etc/systemd/system/tor.service.torcontroller-backup ] \
    && ok "the old unit was moved aside rather than deleted" || bad "the old unit was not retired"

"$BINARY" migrate >/tmp/migrate2.log 2>&1
if grep -q "already up to date" /tmp/migrate2.log && [ "$(grep -c '^TransPort' "$TORRC")" -eq 2 ]; then
    ok "a second migration changes nothing"
else
    bad "migration is not idempotent; postinst runs it on every upgrade"
fi

# ---------------------------------------------------------------------- summary
printf '\n========================================\n'
printf '  passed: %d\n  failed: %d\n' "$PASSED" "$FAILED"
printf '========================================\n'
[ "$FAILED" -eq 0 ]
