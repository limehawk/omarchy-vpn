package main

import (
	"fmt"
	"os"
	"strings"
)

// ConfigInfo holds static config details parsed from a .conf file.
type ConfigInfo struct {
	Address    string
	DNS        string
	Endpoint   string
	PeerKey    string
	AllowedIPs []string // one entry per AllowedIPs line, possibly comma-separated
}

type sheetField struct {
	Label string
	Value string
}

// ParseConfigFile reads a WireGuard .conf file and extracts display fields.
func ParseConfigFile(name string) ConfigInfo {
	if demoMode {
		return demoConfigInfo(name)
	}
	path := fmt.Sprintf("/etc/wireguard/%s.conf", name)
	out, err := helper("cat", name).Output()
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

	name := m.selectedConfig()
	if name == "" {
		return renderKVTable(width, height, nil)
	}

	info := ParseConfigFile(name)
	s := m.vpnStatus[name]
	up := m.isActive(name)
	peer := info.PeerKey
	if len(peer) > 24 {
		peer = peer[:24] + "…"
	}

	fields := []sheetField{
		{"Status", statusWord(up)},
		{"Address", info.Address},
		{"DNS", info.DNS},
		{"Endpoint", firstNonEmpty(s.Endpoint, info.Endpoint)},
		{"Peer", peer},
		{"Path", fmt.Sprintf("/etc/wireguard/%s.conf", name)},
	}
	if up {
		fields = []sheetField{
			{"Status", statusWord(true)},
			{"Address", info.Address},
			{"Endpoint", firstNonEmpty(s.Endpoint, info.Endpoint)},
			{"Download", s.TransferRx},
			{"Upload", s.TransferTx},
			{"Handshake", s.Handshake},
			{"Path", fmt.Sprintf("/etc/wireguard/%s.conf", name)},
		}
	}

	return renderKVTable(width, height, fields)
}

func (m model) renderNetbirdStatusPanel(width, height int) string {
	s := m.netbirdStatus

	switch {
	case s.Connected():
		fields := []sheetField{
			{"Status", statusWord(true)},
			{"Address", s.IP},
			{"FQDN", s.FQDN},
			{"Peers", fmt.Sprintf("%d/%d", s.PeersConnected, s.PeersTotal)},
			{"Download", formatBytes(s.TransferRx)},
			{"Upload", formatBytes(s.TransferTx)},
			{"Management", s.MgmtURL},
		}
		return renderKVTable(width, height, fields)
	case s.DaemonStatus == "":
		return renderKVTable(width, height, []sheetField{
			{"Status", "Daemon down"},
			{"Note", "enable netbird"},
		})
	case s.NeedsLogin():
		return renderKVTable(width, height, []sheetField{
			{"Status", "Login required"},
			{"Note", "run netbird up"},
		})
	default:
		return renderKVTable(width, height, []sheetField{
			{"Status", firstNonEmpty(s.DaemonStatus, statusWord(false))},
			{"Management", s.MgmtURL},
		})
	}
}

func (m model) renderWarpStatusPanel(width, height int) string {
	s := m.warpStatus

	switch {
	case s.Connected():
		return renderKVTable(width, height, []sheetField{
			{"Status", statusWord(true)},
			{"Provider", "Cloudflare WARP"},
		})
	case s.DaemonDown:
		return renderKVTable(width, height, []sheetField{
			{"Status", "Daemon down"},
			{"Provider", "Cloudflare WARP"},
			{"Note", "enable warp-svc"},
		})
	case s.NeedsRegistration():
		return renderKVTable(width, height, []sheetField{
			{"Status", "Login required"},
			{"Note", "warp-cli registration new"},
		})
	case s.Connecting():
		return renderKVTable(width, height, []sheetField{
			{"Status", "Connecting"},
		})
	default:
		return renderKVTable(width, height, []sheetField{
			{"Status", statusWord(false)},
			{"Provider", "Cloudflare WARP"},
		})
	}
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "--"
	}
	return s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func renderSheetField(label, value string) string {
	return dimStyle.Render(label) + "  " + valueStyle.Render(dash(value))
}

func renderKVTable(width, height int, fields []sheetField) string {
	inW := boxInnerWidth(width)
	var lines []string
	if len(fields) == 0 {
		lines = append(lines, formatKVRow("Status", "--", inW))
	}
	for i, f := range fields {
		if i == 1 {
			lines = append(lines, "")
		}
		lines = append(lines, formatKVRow(f.Label, f.Value, inW))
	}
	return titledBox("Details", lines, width, height)
}
