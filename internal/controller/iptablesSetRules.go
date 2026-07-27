package controller

import (
	"fmt"
	"strings"
)

// ApplyIptablesRulesFactory applies the given iptables rules
func (h *CommandHandler) ApplyIptablesRulesFactory(rules []struct {
	Command string
	Args    []string
}) error {
	for _, rule := range rules {
		h.Logger.Printf("[INFO] Applying rule: %s %s", rule.Command, strings.Join(rule.Args, " "))

		// Use CommandRunner instead of exec.Command directly
		output, err := h.CommandRunner.Run("sudo", append([]string{rule.Command}, rule.Args...)...)
		if err != nil {
			h.Logger.Printf("[ERROR] Failed to apply rule: %v. Error: %s", rule, err.Error())
			return fmt.Errorf("failed to apply iptables rule: %w", err)
		}

		h.Logger.Printf("[INFO] Command output: %s", output)
	}
	h.Logger.Println("[INFO] All iptables rules applied successfully.")
	return nil
}

// ClearIptablesRulesFactory removes the given iptables rules
func (h *CommandHandler) ClearIptablesRulesFactory(rules []struct {
	Command string
	Args    []string
}) error {
	for _, rule := range rules {
		h.Logger.Printf("[INFO] Clearing rule: %s %s", rule.Command, strings.Join(rule.Args, " "))

		// Use CommandRunner instead of exec.Command directly
		output, err := h.CommandRunner.Run("sudo", append([]string{rule.Command}, rule.Args...)...)
		if err != nil {
			h.Logger.Printf("[ERROR] Failed to clear rule: %v. Error: %s", rule, err.Error())
			return fmt.Errorf("failed to clear iptables rule: %w", err)
		}

		h.Logger.Printf("[INFO] Command output: %s", output)
	}
	h.Logger.Println("[INFO] All iptables rules cleared successfully.")
	return nil
}

// IPv4 rules for applying and clearing
var Ipv4RulesApply = []struct {
	Command string
	Args    []string
}{
	{"iptables", []string{"-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8118"}},
	{"iptables", []string{"-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", "443", "-j", "REDIRECT", "--to-ports", "8118"}},
}

var Ipv4RulesClear = []struct {
	Command string
	Args    []string
}{
	{"iptables", []string{"-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8118"}},
	{"iptables", []string{"-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "443", "-j", "REDIRECT", "--to-ports", "8118"}},
}

// IPv6 rules for applying and clearing
var Ipv6RulesApply = []struct {
	Command string
	Args    []string
}{
	{"ip6tables", []string{"-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8118"}},
	{"ip6tables", []string{"-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", "443", "-j", "REDIRECT", "--to-ports", "8118"}},
}

var Ipv6RulesClear = []struct {
	Command string
	Args    []string
}{
	{"ip6tables", []string{"-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-ports", "8118"}},
	{"ip6tables", []string{"-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "443", "-j", "REDIRECT", "--to-ports", "8118"}},
}

// IPv6 rules for rejecting traffic
var Ipv6RejectRulesApply = []struct {
	Command string
	Args    []string
}{
	{"ip6tables", []string{"-A", "OUTPUT", "-p", "tcp", "--dport", "9050", "-j", "REJECT"}},
}

var Ipv6RejectRulesClear = []struct {
	Command string
	Args    []string
}{
	{"ip6tables", []string{"-D", "OUTPUT", "-p", "tcp", "--dport", "9050", "-j", "REJECT"}},
}

// ApplyIptablesIPv4 applies IPv4 rules
func (h *CommandHandler) ApplyIptablesIPv4() error {
	return h.ApplyIptablesRulesFactory(Ipv4RulesApply)
}

// ClearIptablesIPv4 clears IPv4 rules
func (h *CommandHandler) ClearIptablesIPv4() error {
	return h.ClearIptablesRulesFactory(Ipv4RulesClear)
}

// ApplyIptablesIPv6 applies IPv6 rules
func (h *CommandHandler) ApplyIptablesIPv6() error {
	return h.ApplyIptablesRulesFactory(Ipv6RulesApply)
}

// ClearIptablesIPv6 clears IPv6 rules
func (h *CommandHandler) ClearIptablesIPv6() error {
	return h.ClearIptablesRulesFactory(Ipv6RulesClear)
}

// ApplyIptablesIPv6Reject applies IPv6 reject rules
func (h *CommandHandler) ApplyIptablesIPv6Reject() error {
	return h.ApplyIptablesRulesFactory(Ipv6RejectRulesApply)
}

// ClearIptablesIPv6Reject clears IPv6 reject rules
func (h *CommandHandler) ClearIptablesIPv6Reject() error {
	return h.ClearIptablesRulesFactory(Ipv6RejectRulesClear)
}
