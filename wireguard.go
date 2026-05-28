package main

import (
	"fmt"
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

// GetActiveVPN returns the first active WireGuard interface that
// corresponds to a config in /etc/wireguard/. Foreign interfaces from
// other WireGuard apps (NetBird, Tailscale, etc.) are ignored.
func GetActiveVPN() string {
	out, err := exec.Command("sudo", "wg", "show", "interfaces").Output()
	if err != nil {
		return ""
	}
	managed := ListConfigs()
	for _, iface := range strings.Fields(string(out)) {
		if isValidConfigName(iface) && slices.Contains(managed, iface) {
			return iface
		}
	}
	return ""
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

func ValidateConfig(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "[Interface]")
}
