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
	Address  string
	DNS      string
	Endpoint string
	PeerKey  string
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
		}
	}
	return info
}

func (m model) renderStatusPanel(width, height int) string {
	innerWidth := width - 4 // border + padding
	contentHeight := height - 2

	var lines []string
	var border lipgloss.Style

	if m.activeVPN != "" {
		border = connectedBorderStyle

		lines = append(lines, "")

		s := m.vpnStatus

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

		info := ParseConfigFile(m.activeVPN)
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
		lines = append(lines, renderField("󰈔", "Config", m.activeVPN, innerWidth))
		lines = append(lines, renderField("󰉋", "Path", fmt.Sprintf("/etc/wireguard/%s.conf", m.activeVPN), innerWidth))

	} else if len(m.configs) > 0 && m.cursor < len(m.configs) {
		// Static preview of highlighted config
		name := m.configs[m.cursor]
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

func renderField(icon, label, value string, maxWidth int) string {
	iconStyled := lipgloss.NewStyle().Foreground(dimCol).Render("  " + icon + " ")
	labelStyled := labelStyle.Render(label)
	valueStyled := valueStyle.Render(value)
	return iconStyled + labelStyled + valueStyled
}
