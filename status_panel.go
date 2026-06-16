package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/lipgloss/v2"
)

// ConfigInfo holds static config details parsed from a .conf file.
type ConfigInfo struct {
	Address    string
	DNS        string
	Endpoint   string
	PeerKey    string
	AllowedIPs []string // one entry per AllowedIPs line, possibly comma-separated
}

// ParseConfigFile reads a WireGuard .conf file and extracts display fields.
func ParseConfigFile(name string) ConfigInfo {
	path := fmt.Sprintf("/etc/wireguard/%s.conf", name)
	out, err := exec.Command("sudo", "cat", path).Output()
	if err != nil {
		data, err2 := os.ReadFile(path)
		if err2 != nil {
			return ConfigInfo{}
		}
		out = data
	}
	data := out

	var info ConfigInfo
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Address":
			info.Address = val
		case "DNS":
			info.DNS = val
		case "Endpoint":
			info.Endpoint = val
		case "PublicKey":
			info.PeerKey = val
		case "AllowedIPs":
			info.AllowedIPs = append(info.AllowedIPs, val)
		}
	}
	return info
}

func (m model) renderStatusPanel(width, height int) string {
	if m.netbirdSelected() {
		return m.renderNetbirdStatusPanel(width, height)
	}
	if m.warpSelected() {
		return m.renderWarpStatusPanel(width, height)
	}

	innerWidth := width - 4 // border + padding
	contentHeight := height - 2

	var lines []string
	var border lipgloss.Style

	if name := m.selectedConfig(); name != "" && m.isActive(name) {
		border = connectedBorderStyle

		lines = append(lines, "")

		s := m.vpnStatus[name]

		// Connection badge
		badge := lipgloss.NewStyle().
			Foreground(green).
			Bold(true).
			Render("  ● Connected")
		lines = append(lines, badge)
		lines = append(lines, "")

		if s.Endpoint != "" {
			lines = append(lines, renderField("󰖟", "Endpoint", s.Endpoint, innerWidth))
		}

		info := ParseConfigFile(name)
		if info.Address != "" {
			lines = append(lines, renderField("󰩟", "Address", info.Address, innerWidth))
		}

		if s.TransferRx != "" {
			lines = append(lines, renderField("↓", "Download", s.TransferRx, innerWidth))
			lines = append(lines, renderField("↑", "Upload", s.TransferTx, innerWidth))
		}

		if s.Handshake != "" {
			lines = append(lines, renderField("󰅐", "Handshake", s.Handshake, innerWidth))
		}

		lines = append(lines, "")
		lines = append(lines, renderField("󰈔", "Config", name, innerWidth))
		lines = append(lines, renderField("󰉋", "Path", fmt.Sprintf("/etc/wireguard/%s.conf", name), innerWidth))

	} else if name != "" {
		// Static preview of highlighted config
		border = inactiveBorderStyle

		info := ParseConfigFile(name)

		lines = append(lines, "")

		badge := dimStyle.Render("  ○ Not connected")
		lines = append(lines, badge)
		lines = append(lines, "")

		if info.Address != "" {
			lines = append(lines, renderField("󰩟", "Address", info.Address, innerWidth))
		}
		if info.DNS != "" {
			lines = append(lines, renderField("󰇖", "DNS", info.DNS, innerWidth))
		}
		if info.Endpoint != "" {
			lines = append(lines, renderField("󰖟", "Endpoint", info.Endpoint, innerWidth))
		}
		if info.PeerKey != "" {
			key := info.PeerKey
			if len(key) > 24 {
				key = key[:24] + "…"
			}
			lines = append(lines, renderField("󰌆", "Peer", key, innerWidth))
		}

		lines = append(lines, "")
		lines = append(lines, renderField("󰈔", "Config", name, innerWidth))
		lines = append(lines, renderField("󰉋", "Path", fmt.Sprintf("/etc/wireguard/%s.conf", name), innerWidth))
	} else {
		border = inactiveBorderStyle
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("  No configs available."))
		lines = append(lines, dimStyle.Render("  Press i to import."))
	}

	// Pad to fill height
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	content := strings.Join(lines, "\n")

	return border.
		Width(width - 2).
		Height(contentHeight).
		Render(content)
}

func (m model) renderNetbirdStatusPanel(width, height int) string {
	innerWidth := width - 4
	contentHeight := height - 2

	var lines []string
	var border lipgloss.Style
	s := m.netbirdStatus

	lines = append(lines, "")

	switch {
	case s.Connected():
		border = connectedBorderStyle
		badge := lipgloss.NewStyle().
			Foreground(green).
			Bold(true).
			Render("  ● Connected")
		lines = append(lines, badge, "")
		if s.IP != "" {
			lines = append(lines, renderField("󰩟", "Address", s.IP, innerWidth))
		}
		if s.FQDN != "" {
			lines = append(lines, renderField("󰖟", "FQDN", s.FQDN, innerWidth))
		}
		lines = append(lines, renderField("󰀂", "Peers", fmt.Sprintf("%d/%d connected", s.PeersConnected, s.PeersTotal), innerWidth))
		lines = append(lines, renderField("↓", "Download", formatBytes(s.TransferRx), innerWidth))
		lines = append(lines, renderField("↑", "Upload", formatBytes(s.TransferTx), innerWidth))
		if s.MgmtURL != "" {
			lines = append(lines, "")
			lines = append(lines, renderField("󰖟", "Management", s.MgmtURL, innerWidth))
		}

	case s.DaemonStatus == "":
		border = inactiveBorderStyle
		lines = append(lines, dimStyle.Render("  ○ Daemon not running"), "")
		lines = append(lines, dimStyle.Render("  sudo systemctl enable --now netbird"))

	case s.NeedsLogin():
		border = inactiveBorderStyle
		lines = append(lines, warnStyle.Render("  ○ Login required"), "")
		lines = append(lines, dimStyle.Render("  Run `netbird up` in a terminal to log in."))

	default: // Idle, Connecting, or unknown future states
		border = inactiveBorderStyle
		lines = append(lines, dimStyle.Render("  ○ Not connected"), "")
		lines = append(lines, renderField("󰒓", "Status", s.DaemonStatus, innerWidth))
		if s.MgmtURL != "" {
			lines = append(lines, renderField("󰖟", "Management", s.MgmtURL, innerWidth))
		}
	}

	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	return border.
		Width(width - 2).
		Height(contentHeight).
		Render(strings.Join(lines, "\n"))
}

func (m model) renderWarpStatusPanel(width, height int) string {
	innerWidth := width - 4
	contentHeight := height - 2

	var lines []string
	var border lipgloss.Style
	s := m.warpStatus

	lines = append(lines, "")

	switch {
	case s.Connected():
		border = connectedBorderStyle
		badge := lipgloss.NewStyle().
			Foreground(green).
			Bold(true).
			Render("  ● Connected")
		lines = append(lines, badge, "")
		lines = append(lines, renderField("󰖟", "Provider", "Cloudflare WARP", innerWidth))

	case s.DaemonDown:
		border = inactiveBorderStyle
		lines = append(lines, dimStyle.Render("  ○ Daemon not running"), "")
		lines = append(lines, dimStyle.Render("  sudo systemctl enable --now warp-svc"))

	case s.NeedsRegistration():
		border = inactiveBorderStyle
		lines = append(lines, warnStyle.Render("  ○ Registration required"), "")
		lines = append(lines, dimStyle.Render("  Run `warp-cli registration new`, or enroll via"))
		lines = append(lines, dimStyle.Render("  your Zero Trust org, in a terminal."))

	case s.Connecting():
		border = inactiveBorderStyle
		lines = append(lines, dimStyle.Render("  ○ Connecting…"))

	default: // Disconnected, No network, or unknown future states
		border = inactiveBorderStyle
		lines = append(lines, dimStyle.Render("  ○ Not connected"), "")
		lines = append(lines, dimStyle.Render("  Press enter to connect."))
	}

	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	return border.
		Width(width - 2).
		Height(contentHeight).
		Render(strings.Join(lines, "\n"))
}

func renderField(icon, label, value string, maxWidth int) string {
	iconStyled := lipgloss.NewStyle().Foreground(dimCol).Render("  " + icon + " ")
	labelStyled := labelStyle.Render(label)
	valueStyled := valueStyle.Render(value)
	return iconStyled + labelStyled + valueStyled
}
