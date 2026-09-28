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
			helpTitleStyle.Render("Keyboard shortcuts") + "\n\n" +
				helpView + "\n\n" +
				dimStyle.Render("omarchy-vpn "+displayVersion()+" · any key closes"),
		)
		v := tea.NewView(lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			overlay,
		))
		v.AltScreen = true
		return v
	}

	if m.modal == modalHookConfirm {
		var b strings.Builder
		for _, h := range m.importHooks {
			b.WriteString("  " + warnStyle.Render(h) + "\n")
		}
		overlay := helpOverlayStyle.Render(
			helpTitleStyle.Render("Import "+m.importName+"?") + "\n\n" +
				"This config runs these commands " + errorStyle.Render("as root") + " every time it connects:\n\n" +
				b.String() + "\n" +
				"Only continue if you trust whoever gave you this file.\n" +
				dimStyle.Render("y import (asks for your password) · any other key cancels"),
		)
		v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay))
		v.AltScreen = true
		return v
	}

	// header (2) + blank + panes + blank + footer
	innerW := pageInnerWidth(m.width)
	titleBar := m.renderTitleBar()

	var bottom string
	switch {
	case m.importRename():
		bottom = warnStyle.Render("  Enter a name (max 15, must include a letter)")
	case m.message != "" && time.Now().Before(m.messageExp):
		bottom = m.message
	default:
		m.message = ""
		bottom = renderKeyFooter(m.help, m.keys.ShortHelp(), innerW)
	}

	avail := m.height - 2 - 1 - 1 - max(1, lipgloss.Height(bottom))
	if avail < 6 {
		avail = 6
	}
	panels := m.renderPanes(innerW, avail)

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		"",
		panels,
		"",
		bottom,
	)
	if pad := m.width - innerW; pad > 0 {
		body = indentBlock(body, pad/2)
	}

	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func (m model) renderPanes(innerW, avail int) string {
	if useStackedLayout(m.width, m.height) {
		topH, botH := stackPaneHeights(avail, m.tunnelContentLines())
		return lipgloss.JoinVertical(lipgloss.Left,
			m.renderConfigPanel(innerW, topH),
			"",
			m.renderStatusPanel(innerW, botH),
		)
	}

	leftWidth := (innerW - paneGap) * 2 / 5
	rightWidth := innerW - paneGap - leftWidth
	if leftWidth < 32 {
		leftWidth = 32
		rightWidth = innerW - paneGap - leftWidth
	}
	if rightWidth < 20 {
		rightWidth = 20
		leftWidth = innerW - paneGap - rightWidth
	}
	left := m.renderConfigPanel(leftWidth, avail)
	right := m.renderStatusPanel(rightWidth, avail)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", paneGap), right)
}

func (m model) tunnelContentLines() int {
	n := m.listLen()
	if n == 0 {
		return 1
	}
	return n*2 + (n - 1)
}

func (m model) sessionConnected() bool {
	return len(m.activeVPNs) > 0 || m.netbirdStatus.Connected() || m.warpStatus.Connected()
}

func (m model) sessionNames() []string {
	names := append([]string{}, m.activeVPNs...)
	if m.netbirdStatus.Connected() {
		names = append(names, "NetBird")
	}
	if m.warpStatus.Connected() {
		names = append(names, "Cloudflare WARP")
	}
	return names
}

func (m model) renderTitleBar() string {
	w := pageInnerWidth(m.width)
	name := itemStyle.Render("omarchy-vpn")
	ver := dimStyle.Render(displayVersion())
	pad := w - lipgloss.Width(name) - lipgloss.Width(ver)
	if pad < 1 {
		pad = 1
	}
	line1 := name + strings.Repeat(" ", pad) + ver

	st := statusWord(m.sessionConnected())
	var line2 string
	if m.sessionConnected() {
		line2 = connectedStyle.Render(st)
		if names := strings.Join(m.sessionNames(), "  "); names != "" {
			line2 += "  " + dimStyle.Render(names)
		}
	} else {
		line2 = dimStyle.Render(st)
	}
	return line1 + "\n" + line2
}
