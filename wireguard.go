package main

import (
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// isValidConfigName returns true if name contains only [a-zA-Z0-9_-].
func isValidConfigName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

type VPNStatus struct {
	Interface  string
	Endpoint   string
	TransferRx string
	TransferTx string
	Handshake  string
}

// GetActiveVPNs returns all active WireGuard interfaces that
// correspond to configs in /etc/wireguard/. Foreign interfaces from
// other WireGuard apps (NetBird, Tailscale, etc.) are ignored.
func GetActiveVPNs() []string {
	out, err := exec.Command("sudo", "wg", "show", "interfaces").Output()
	if err != nil {
		return nil
	}
	managed := ListConfigs()
	var active []string
	for _, iface := range strings.Fields(string(out)) {
		if isValidConfigName(iface) && slices.Contains(managed, iface) {
			active = append(active, iface)
		}
	}
	return active
}

func ListConfigs() []string {
	out, err := exec.Command("sudo", "ls", "/etc/wireguard").Output()
	if err != nil {
		return nil
	}
	var configs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasSuffix(line, ".conf") {
			name := strings.TrimSuffix(line, ".conf")
			if isValidConfigName(name) {
				configs = append(configs, name)
			}
		}
	}
	return configs
}

func ConnectVPN(name string) error {
	out, err := exec.Command("sudo", "wg-quick", "up", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", extractError(string(out), err))
	}
	return nil
}

func DisconnectVPN(name string) error {
	out, err := exec.Command("sudo", "wg-quick", "down", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", extractError(string(out), err))
	}
	return nil
}

// extractError pulls the meaningful error line from wg-quick output,
// skipping the [#] command trace lines.
func extractError(output string, fallback error) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.HasPrefix(line, "[#]") {
			return line
		}
	}
	return fallback.Error()
}

func GetVPNStatus(name string) (VPNStatus, error) {
	out, err := exec.Command("sudo", "wg", "show", name).Output()
	if err != nil {
		return VPNStatus{}, err
	}
	status := VPNStatus{Interface: name}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "endpoint:"):
			status.Endpoint = strings.TrimSpace(strings.TrimPrefix(line, "endpoint:"))
		case strings.HasPrefix(line, "transfer:"):
			parts := strings.TrimSpace(strings.TrimPrefix(line, "transfer:"))
			fields := strings.Split(parts, ",")
			if len(fields) >= 1 {
				status.TransferRx = strings.TrimSpace(fields[0])
			}
			if len(fields) >= 2 {
				status.TransferTx = strings.TrimSpace(fields[1])
			}
		case strings.HasPrefix(line, "latest handshake:"):
			status.Handshake = strings.TrimSpace(strings.TrimPrefix(line, "latest handshake:"))
		}
	}
	return status, nil
}

func ImportConfig(src, name string) error {
	if err := exec.Command("sudo", "cp", src, fmt.Sprintf("/etc/wireguard/%s.conf", name)).Run(); err != nil {
		return err
	}
	return exec.Command("sudo", "chmod", "600", fmt.Sprintf("/etc/wireguard/%s.conf", name)).Run()
}

func RemoveConfig(name string) error {
	return exec.Command("sudo", "rm", fmt.Sprintf("/etc/wireguard/%s.conf", name)).Run()
}

func RenameConfig(oldName, newName string) error {
	oldPath := fmt.Sprintf("/etc/wireguard/%s.conf", oldName)
	newPath := fmt.Sprintf("/etc/wireguard/%s.conf", newName)
	out, err := exec.Command("sudo", "mv", oldPath, newPath).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}

// parseAllowedIPs converts raw AllowedIPs values (each possibly a
// comma-separated list) into masked prefixes. Bare addresses become
// single-address prefixes; unparseable entries are skipped.
func parseAllowedIPs(vals []string) []netip.Prefix {
	var prefixes []netip.Prefix
	for _, v := range vals {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if p, err := netip.ParsePrefix(part); err == nil {
				prefixes = append(prefixes, p.Masked())
			} else if a, err := netip.ParseAddr(part); err == nil {
				prefixes = append(prefixes, netip.PrefixFrom(a, a.BitLen()))
			}
		}
	}
	return prefixes
}

// allowedIPsOverlap reports whether two AllowedIPs sets route any of the
// same address space. Configs whose AllowedIPs are missing or entirely
// unparseable are treated as overlapping, so the safe switch behavior
// (disconnect first) applies when routes can't be compared.
func allowedIPsOverlap(a, b []string) bool {
	pa, pb := parseAllowedIPs(a), parseAllowedIPs(b)
	if len(pa) == 0 || len(pb) == 0 {
		return true
	}
	for _, x := range pa {
		for _, y := range pb {
			if x.Overlaps(y) {
				return true
			}
		}
	}
	return false
}

// conflictingVPNs returns the active tunnels whose AllowedIPs overlap the
// named config's AllowedIPs. These must come down before the config can go
// up; tunnels with disjoint routes can stay connected alongside it.
func conflictingVPNs(name string, active []string) []string {
	target := ParseConfigFile(name).AllowedIPs
	var conflicts []string
	for _, a := range active {
		if allowedIPsOverlap(target, ParseConfigFile(a).AllowedIPs) {
			conflicts = append(conflicts, a)
		}
	}
	return conflicts
}

func ValidateConfig(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "[Interface]")
}
