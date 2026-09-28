package main

func (m model) importRename() bool {
	return m.modal == modalRenaming && m.importPath != ""
}

func (m model) renderConfigPanel(width, height int) string {
	inW := boxInnerWidth(width)
	var lines []string
	if m.listLen() == 0 && !m.importRename() {
		lines = append(lines, dimStyle.Render("No configs."))
	} else {
		for _, r := range m.indexRows() {
			lines = append(lines, m.renderIndexItem(r, inW)...)
		}
		if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = lines[:n-1]
		}
	}
	if m.importRename() {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, m.renameInput.View(), dimStyle.Render("max 15 characters, include a letter"))
	}
	return titledBox("Tunnels", lines, width, height)
}

type indexRow struct {
	name      string
	kind      string
	connected bool
	index     int
}

func statusWord(connected bool) string {
	if connected {
		return "Connected"
	}
	return "Disconnected"
}

func (m model) indexRows() []indexRow {
	var rows []indexRow
	idx := 0
	if m.netbirdAvail {
		rows = append(rows, indexRow{netbirdRowName, "NetBird", m.netbirdStatus.Connected(), idx})
		idx++
	}
	if m.warpAvail {
		rows = append(rows, indexRow{warpRowName, "WARP", m.warpStatus.Connected(), idx})
		idx++
	}
	for _, cfg := range m.configs {
		rows = append(rows, indexRow{cfg, "WireGuard", m.isActive(cfg), idx})
		idx++
	}
	return rows
}

func (m model) renderIndexItem(row indexRow, width int) []string {
	selected := row.index == m.cursor
	if selected && m.modal == modalRenaming && m.importPath == "" {
		return []string{m.renameInput.View(), ""}
	}
	if selected && m.modal == modalDeleting {
		return []string{
			errorStyle.Render("Delete "+row.name+"?") + " " + dimStyle.Render("[y/n]"),
			"",
		}
	}

	name := truncate(row.name, width)
	if selected {
		name = selectedNameStyle.Render(name)
	} else {
		name = itemStyle.Render(name)
	}

	st := statusWord(row.connected)
	if selected && m.modal == modalConnecting && m.connectName == row.name {
		st = m.spinner.View()
	}
	meta := row.kind + " · " + st
	var metaLine string
	if row.connected && m.modal != modalConnecting {
		metaLine = connectedStyle.Render(truncate(meta, width))
	} else {
		metaLine = dimStyle.Render(truncate(meta, width))
	}
	return []string{name, metaLine, ""}
}
