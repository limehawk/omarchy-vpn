package main

import (
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// helperPath is the root helper installed by the package. The sudoers file
// allows only this command; it takes config names, never paths, and refuses
// imports that carry PreUp/PostUp/PreDown/PostDown commands.
const helperPath = "/usr/lib/omarchy-vpn/helper"

func helper(args ...string) *exec.Cmd {
	return exec.Command("sudo", append([]string{helperPath}, args...)...)
}

// helperError prefers the helper's own message over "exit status 1".
func helperError(out []byte, err error) error {
	if msg := strings.TrimSpace(string(out)); msg != "" {
		return fmt.Errorf("%s", msg)
	}
	return err
}

// maxInterfaceName is wg-quick's interface-name limit (Linux IFNAMSIZ-1).
// Names longer than this are treated as a filesystem path, which produces
// the misleading "`name' does not exist" instead of looking up
// /etc/wireguard/<name>.conf.
const maxInterfaceName = 15

// isValidConfigName returns true if name contains only [a-zA-Z0-9_-] and
// does not start with "-" (it would be read as an option).
// Length and "all digits" are not checked here so already-imported
// invalid names still appear in the list and can be renamed.
func isValidConfigName(name string) bool {
	if name == "" || name[0] == '-' {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func isDigitsOnly(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// interfaceNameError is nil when name is safe to pass to wg-quick as an
// interface (charset, max 15 characters, not all digits).
func interfaceNameError(name string) error {
	if !isValidConfigName(name) {
		return fmt.Errorf("invalid config name")
	}
	if len(name) > maxInterfaceName {
		return fmt.Errorf("name %q is too long for WireGuard (max %d characters); rename it first", name, maxInterfaceName)
	}
	if isDigitsOnly(name) {
		return fmt.Errorf("name %q is only numbers; systemd treats that as an interface index", name)
	}
	return nil
}

// suggestInterfaceName turns a raw config name into something wg-quick and
// systemd-resolved will accept: charset-safe, ≤15 characters, not all digits.
func suggestInterfaceName(name string) string {
	name = strings.TrimLeft(sanitizeName(name), "-")
	if interfaceNameError(name) == nil {
		return name
	}
	if isDigitsOnly(name) {
		name = "wg" + name
	}
	if len(name) > maxInterfaceName {
		name = strings.TrimRight(name[:maxInterfaceName], "-_")
	}
	if interfaceNameError(name) != nil && isDigitsOnly(name) {
		name = "wg" + name
		if len(name) > maxInterfaceName {
			name = strings.TrimRight(name[:maxInterfaceName], "-_")
		}
	}
	if interfaceNameError(name) != nil {
		return "imported"
	}
	return name
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
	if demoMode {
		return demoActiveVPNs()
	}
	out, err := helper("interfaces").Output()
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
	if demoMode {
		return demoListConfigs()
	}
	out, err := helper("list").Output()
	if err != nil {
		return nil
	}
	var configs []string
	for _, name := range strings.Fields(string(out)) {
		if isValidConfigName(name) {
			configs = append(configs, name)
		}
	}
	return configs
}

func ConnectVPN(name string) error {
	if err := interfaceNameError(name); err != nil {
		return err
	}
	if demoMode {
		demoConnect(name)
		return nil
	}
	out, err := helper("up", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", extractError(string(out), err))
	}
	return nil
}

func DisconnectVPN(name string) error {
	if demoMode {
		demoDisconnect(name)
		return nil
	}
	out, err := helper("down", name).CombinedOutput()
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
	if demoMode {
		return demoVPNStatus(name), nil
	}
	out, err := helper("show", name).Output()
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
	if err := interfaceNameError(name); err != nil {
		return err
	}
	if demoMode {
		return nil
	}
	// The file is opened as the user and streamed in, so the helper never
	// touches a path the user chose.
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	cmd := helper("import", name)
	cmd.Stdin = f
	out, err := cmd.CombinedOutput()
	if err != nil {
		return helperError(out, err)
	}
	return nil
}

func RemoveConfig(name string) error {
	if demoMode {
		return nil
	}
	out, err := helper("rm", name).CombinedOutput()
	if err != nil {
		return helperError(out, err)
	}
	return nil
}

func RenameConfig(oldName, newName string) error {
	if err := interfaceNameError(newName); err != nil {
		return err
	}
	if demoMode {
		return nil
	}
	out, err := helper("rename", oldName, newName).CombinedOutput()
	if err != nil {
		return helperError(out, err)
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
