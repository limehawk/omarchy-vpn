package main

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("")
	}

	// File picker
	if m.modal == modalImporting {
		v := tea.NewView(m.filePicker.View())
		v.AltScreen = true
		return v
	}

	// Help overlay replaces everything
	if m.modal == modalHelp {
		helpView := m.help.View(m.keys)
		overlay := helpOverlayStyle.Render(
			helpTitleStyle.Render("󰋖  Keyboard Shortcuts") + "\n\n" +
				helpView + "\n\n" +
				dimStyle.Render("omarchy-vpn "+displayVersion()+" · Press any key to close"),
		)
		v := tea.NewView(lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			overlay,
		))
		v.AltScreen = true
		return v
	}

	// Layout: title bar (1) + gap (1) + panels + gap (1) + bottom bar (1)
	titleBar := m.renderTitleBar()
	titleHeight := 2  // title + gap
	bottomHeight := 2 // gap + shortcuts
	panelHeight := m.height - titleHeight - bottomHeight
	if panelHeight < 5 {
		panelHeight = 5
	}

	// Panel widths: 40/60 split
	leftWidth := m.width * 2 / 5
	rightWidth := m.width - leftWidth

	if leftWidth < 24 {
		leftWidth = 24
		rightWidth = m.width - leftWidth
	}

	// Render panels
	left := m.renderConfigPanel(leftWidth, panelHeight)
	right := m.renderStatusPanel(rightWidth, panelHeight)

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// Bottom bar
	var bottom string
	if m.message != "" && time.Now().Before(m.messageExp) {
		bottom = " " + m.message
	} else {
		m.message = ""
		bottom = " " + m.help.View(m.keys)
	}

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		panels,
		bottom,
	))
	v.AltScreen = true
	return v
}

func (m model) renderTitleBar() string {
	// Mirror the waybar tray glyph: shield-check when up, shield-off when down.
	glyph := "󰳌"
	if len(m.activeVPNs) > 0 || m.netbirdStatus.Connected() || m.warpStatus.Connected() {
		glyph = "󰦝"
	}
	icon := titleStyle.Render(glyph + " ")
	name := titleStyle.Render("omarchy-vpn")
	ver := titleAccentStyle.Render(" " + displayVersion())

	sep := titleAccentStyle.Render("  ─  ")
	var badges []string
	for _, vpn := range m.activeVPNs {
		badges = append(badges, connectedStyle.Render("● "+vpn))
	}
	if m.netbirdStatus.Connected() {
		badges = append(badges, connectedStyle.Render("● NetBird"))
	}
	if m.warpStatus.Connected() {
		badges = append(badges, connectedStyle.Render("● Cloudflare WARP"))
	}
	var status string
	if len(badges) > 0 {
		status = sep + strings.Join(badges, "  ")
	} else {
		status = sep + dimStyle.Render("○ disconnected")
	}

	return " " + icon + name + ver + status + "\n"
}
