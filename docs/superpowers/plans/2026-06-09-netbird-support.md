# NetBird Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make NetBird a first-class citizen in omarchy-vpn — a pinned row in the config list that can be toggled with `netbird up`/`netbird down`, with status panel, title bar, and waybar awareness.

**Architecture:** New `netbird.go` backend mirrors `wireguard.go` (exec wrappers + status parsing from `netbird status --json`). The model gains `netbirdAvail`/`netbirdStatus` fields and three cursor helpers that translate the unified list index (NetBird row pinned at top) to configs. NetBird coexists with WireGuard tunnels — toggling one never touches the other. No sudo for netbird commands (CLI talks to its daemon over a socket).

**Tech Stack:** Go, Bubble Tea v2, Lip Gloss v2. Verified against netbird v0.72.1 JSON schema (`daemonStatus`, `netbirdIp`, `fqdn`, `management`, `peers.details[].transferReceived/transferSent`).

**Spec:** `docs/superpowers/specs/2026-06-09-netbird-support-design.md`

---

### Task 1: NetBird status parsing (backend)

**Files:**
- Create: `netbird.go`
- Create: `netbird_test.go`

- [ ] **Step 1: Write the failing tests**

Create `netbird_test.go`:

```go
package main

import "testing"

const netbirdStatusFixture = `{
  "peers": {
    "total": 3,
    "connected": 2,
    "details": [
      {"fqdn": "peer-a.netbird.cloud", "netbirdIp": "100.92.0.2", "status": "Connected", "transferReceived": 1024, "transferSent": 2048},
      {"fqdn": "peer-b.netbird.cloud", "netbirdIp": "100.92.0.3", "status": "Connected", "transferReceived": 100, "transferSent": 200},
      {"fqdn": "peer-c.netbird.cloud", "netbirdIp": "100.92.0.4", "status": "Disconnected", "transferReceived": 0, "transferSent": 0}
    ]
  },
  "cliVersion": "0.72.1",
  "daemonVersion": "0.72.1",
  "daemonStatus": "Connected",
  "management": {"url": "https://api.netbird.io:443", "connected": true, "error": ""},
  "signal": {"url": "https://signal.netbird.io:443", "connected": true, "error": ""},
  "netbirdIp": "100.92.0.1/16",
  "publicKey": "abc123=",
  "usesKernelInterface": true,
  "fqdn": "mybox.netbird.cloud"
}`

func TestParseNetBirdStatus(t *testing.T) {
	s, err := parseNetBirdStatus([]byte(netbirdStatusFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Connected() {
		t.Error("expected Connected() to be true")
	}
	if s.IP != "100.92.0.1/16" {
		t.Errorf("IP = %q, want 100.92.0.1/16", s.IP)
	}
	if s.FQDN != "mybox.netbird.cloud" {
		t.Errorf("FQDN = %q", s.FQDN)
	}
	if s.MgmtURL != "https://api.netbird.io:443" || !s.MgmtConnected {
		t.Errorf("management = %q/%v", s.MgmtURL, s.MgmtConnected)
	}
	if s.PeersConnected != 2 || s.PeersTotal != 3 {
		t.Errorf("peers = %d/%d, want 2/3", s.PeersConnected, s.PeersTotal)
	}
	if s.TransferRx != 1124 || s.TransferTx != 2248 {
		t.Errorf("transfer = %d/%d, want 1124/2248", s.TransferRx, s.TransferTx)
	}
}

func TestParseNetBirdStatusNeedsLogin(t *testing.T) {
	s, err := parseNetBirdStatus([]byte(`{"daemonStatus": "NeedsLogin"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Connected() {
		t.Error("expected Connected() to be false")
	}
	if !s.NeedsLogin() {
		t.Error("expected NeedsLogin() to be true")
	}
}

func TestParseNetBirdStatusInvalid(t *testing.T) {
	if _, err := parseNetBirdStatus([]byte("not json")); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestNeedsLoginStates(t *testing.T) {
	for _, st := range []string{"NeedsLogin", "LoginFailed", "SessionExpired"} {
		if !(NetBirdStatus{DaemonStatus: st}).NeedsLogin() {
			t.Errorf("NeedsLogin() false for %s", st)
		}
	}
	for _, st := range []string{"Connected", "Idle", "Connecting", ""} {
		if (NetBirdStatus{DaemonStatus: st}).NeedsLogin() {
			t.Errorf("NeedsLogin() true for %q", st)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{5368709120, "5.0 GiB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.in); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./... -run 'NetBird|FormatBytes' -v`
Expected: compile error — `parseNetBirdStatus`, `NetBirdStatus`, `formatBytes` undefined.

- [ ] **Step 3: Write the implementation**

Create `netbird.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// netbirdRowName is the display name and connectName sentinel for the
// pinned NetBird list row.
const netbirdRowName = "NetBird"

// Daemon states reported by `netbird status --json` (daemonStatus field).
const (
	netbirdConnected      = "Connected"
	netbirdNeedsLogin     = "NeedsLogin"
	netbirdLoginFailed    = "LoginFailed"
	netbirdSessionExpired = "SessionExpired"
)

// NetBirdStatus holds the fields we display from `netbird status --json`.
type NetBirdStatus struct {
	DaemonStatus   string
	IP             string
	FQDN           string
	MgmtURL        string
	MgmtConnected  bool
	PeersConnected int
	PeersTotal     int
	TransferRx     int64
	TransferTx     int64
}

func (s NetBirdStatus) Connected() bool { return s.DaemonStatus == netbirdConnected }

// NeedsLogin reports whether `netbird up` would block on a browser SSO flow.
func (s NetBirdStatus) NeedsLogin() bool {
	switch s.DaemonStatus {
	case netbirdNeedsLogin, netbirdLoginFailed, netbirdSessionExpired:
		return true
	}
	return false
}

// NetBirdAvailable reports whether the netbird CLI is installed.
func NetBirdAvailable() bool {
	_, err := exec.LookPath("netbird")
	return err == nil
}

func parseNetBirdStatus(data []byte) (NetBirdStatus, error) {
	var raw struct {
		DaemonStatus string `json:"daemonStatus"`
		IP           string `json:"netbirdIp"`
		FQDN         string `json:"fqdn"`
		Management   struct {
			URL       string `json:"url"`
			Connected bool   `json:"connected"`
		} `json:"management"`
		Peers struct {
			Total     int `json:"total"`
			Connected int `json:"connected"`
			Details   []struct {
				TransferReceived int64 `json:"transferReceived"`
				TransferSent     int64 `json:"transferSent"`
			} `json:"details"`
		} `json:"peers"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return NetBirdStatus{}, err
	}
	s := NetBirdStatus{
		DaemonStatus:   raw.DaemonStatus,
		IP:             raw.IP,
		FQDN:           raw.FQDN,
		MgmtURL:        raw.Management.URL,
		MgmtConnected:  raw.Management.Connected,
		PeersConnected: raw.Peers.Connected,
		PeersTotal:     raw.Peers.Total,
	}
	for _, p := range raw.Peers.Details {
		s.TransferRx += p.TransferReceived
		s.TransferTx += p.TransferSent
	}
	return s, nil
}

// GetNetBirdStatus queries the daemon. A zero-value status (DaemonStatus "")
// means the daemon is unreachable.
func GetNetBirdStatus() (NetBirdStatus, error) {
	out, err := exec.Command("netbird", "status", "--json").Output()
	if err != nil {
		return NetBirdStatus{}, err
	}
	return parseNetBirdStatus(out)
}

func NetBirdUp() error {
	out, err := exec.Command("netbird", "up").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", extractError(string(out), err))
	}
	return nil
}

func NetBirdDown() error {
	out, err := exec.Command("netbird", "down").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", extractError(string(out), err))
	}
	return nil
}

// formatBytes renders a byte count in binary units (KiB, MiB, ...).
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... -run 'NetBird|FormatBytes' -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add netbird.go netbird_test.go
git commit -m "feat: NetBird backend — status parsing, up/down wrappers"
```

---

### Task 2: Model state + cursor helpers

**Files:**
- Modify: `model.go` (struct fields, messages, `initialModel`, `statusTickMsg` handler)
- Create: `model_test.go`

- [ ] **Step 1: Write the failing tests**

Create `model_test.go`:

```go
package main

import "testing"

func TestCursorHelpersWithNetbird(t *testing.T) {
	m := model{netbirdAvail: true, configs: []string{"alpha", "beta"}}

	if got := m.listLen(); got != 3 {
		t.Errorf("listLen() = %d, want 3", got)
	}

	m.cursor = 0
	if !m.netbirdSelected() {
		t.Error("cursor 0 should select NetBird row")
	}
	if got := m.selectedConfig(); got != "" {
		t.Errorf("selectedConfig() on NetBird row = %q, want \"\"", got)
	}

	m.cursor = 1
	if m.netbirdSelected() {
		t.Error("cursor 1 should not select NetBird row")
	}
	if got := m.selectedConfig(); got != "alpha" {
		t.Errorf("selectedConfig() = %q, want alpha", got)
	}

	m.cursor = 2
	if got := m.selectedConfig(); got != "beta" {
		t.Errorf("selectedConfig() = %q, want beta", got)
	}
}

func TestCursorHelpersWithoutNetbird(t *testing.T) {
	m := model{netbirdAvail: false, configs: []string{"alpha", "beta"}}

	if got := m.listLen(); got != 2 {
		t.Errorf("listLen() = %d, want 2", got)
	}
	if m.netbirdSelected() {
		t.Error("netbirdSelected() must be false when netbird unavailable")
	}
	m.cursor = 0
	if got := m.selectedConfig(); got != "alpha" {
		t.Errorf("selectedConfig() = %q, want alpha", got)
	}
}

func TestSelectedConfigEmptyList(t *testing.T) {
	m := model{netbirdAvail: true}
	m.cursor = 0
	if got := m.selectedConfig(); got != "" {
		t.Errorf("selectedConfig() = %q, want \"\"", got)
	}
	if got := m.listLen(); got != 1 {
		t.Errorf("listLen() = %d, want 1", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./... -run Cursor -v`
Expected: compile error — `netbirdAvail`, `listLen`, etc. undefined.

- [ ] **Step 3: Add model fields, messages, and helpers**

In `model.go`, add to the `model` struct after the `vpnStatus VPNStatus` field:

```go
	// NetBird state (only meaningful when netbirdAvail is true)
	netbirdAvail  bool
	netbirdStatus NetBirdStatus
```

Add a message type after `disconnectDoneMsg`:

```go
type netbirdDoneMsg struct {
	up  bool // true for `netbird up`, false for `netbird down`
	err error
}
```

Add the cursor helpers after the `model` struct definition:

```go
// listLen returns the number of rows in the left panel list, including the
// pinned NetBird row when present.
func (m model) listLen() int {
	n := len(m.configs)
	if m.netbirdAvail {
		n++
	}
	return n
}

// netbirdSelected reports whether the cursor is on the pinned NetBird row.
func (m model) netbirdSelected() bool {
	return m.netbirdAvail && m.cursor == 0
}

// selectedConfig returns the WireGuard config under the cursor, or "" if the
// cursor is on the NetBird row or the list is empty.
func (m model) selectedConfig() string {
	i := m.cursor
	if m.netbirdAvail {
		i--
	}
	if i < 0 || i >= len(m.configs) {
		return ""
	}
	return m.configs[i]
}
```

In `initialModel()`, after `m.configs = ListConfigs()`:

```go
	m.netbirdAvail = NetBirdAvailable()
	if m.netbirdAvail {
		m.netbirdStatus, _ = GetNetBirdStatus()
	}
```

In the `statusTickMsg` case, replace the body with:

```go
	case statusTickMsg:
		if m.activeVPN != "" {
			m.vpnStatus, _ = GetVPNStatus(m.activeVPN)
		}
		if m.netbirdAvail {
			m.netbirdStatus, _ = GetNetBirdStatus()
		}
		configs := ListConfigs()
		if len(configs) != len(m.configs) {
			m.configs = configs
			if m.cursor >= m.listLen() && m.cursor > 0 {
				m.cursor = m.listLen() - 1
			}
		}
		return m, statusTick()
```

Add a `netbirdDoneMsg` handler after the `disconnectDoneMsg` case:

```go
	case netbirdDoneMsg:
		m.modal = modalNone
		if m.netbirdAvail {
			m.netbirdStatus, _ = GetNetBirdStatus()
		}
		if msg.err != nil {
			m.setMessage(errorStyle.Render("  NetBird: " + msg.err.Error()))
		} else if msg.up {
			m.setMessage(connectedStyle.Render("  NetBird connected"))
		} else {
			m.setMessage(dimStyle.Render("  NetBird disconnected"))
		}
		return m, nil
```

- [ ] **Step 4: Run tests + build**

Run: `go test ./... -run Cursor -v && go build .`
Expected: tests PASS, build clean.

- [ ] **Step 5: Commit**

```bash
git add model.go model_test.go
git commit -m "feat: model state and cursor helpers for NetBird row"
```

---

### Task 3: Key routing — connect, disconnect, rename/delete guards

**Files:**
- Modify: `model.go` (`updateNormal`, `updateDelete`)

- [ ] **Step 1: Update navigation bounds in `updateNormal`**

Replace the `Down` and `Up` cases:

```go
	case key.Matches(msg, m.keys.Down):
		if m.listLen() > 0 {
			m.cursor++
			if m.cursor >= m.listLen() {
				m.cursor = 0
			}
		}

	case key.Matches(msg, m.keys.Up):
		if m.listLen() > 0 {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = m.listLen() - 1
			}
		}
```

- [ ] **Step 2: Route Connect through the NetBird row**

Replace the `Connect` case body:

```go
	case key.Matches(msg, m.keys.Connect):
		if m.netbirdSelected() {
			return m.connectNetbird()
		}
		selected := m.selectedConfig()
		if selected == "" {
			break
		}
		if selected == m.activeVPN {
			m.setMessage(dimStyle.Render("  Already connected"))
			break
		}
		m.modal = modalConnecting
		m.connectName = selected
		activeVPN := m.activeVPN
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			if activeVPN != "" {
				if err := DisconnectVPN(activeVPN); err != nil {
					return connectDoneMsg{name: selected, err: err}
				}
			}
			err := ConnectVPN(selected)
			return connectDoneMsg{name: selected, err: err}
		})
```

Add `connectNetbird` as a new method after `updateNormal`:

```go
func (m *model) connectNetbird() (tea.Model, tea.Cmd) {
	if m.netbirdStatus.Connected() {
		m.setMessage(dimStyle.Render("  Already connected"))
		return m, nil
	}
	if m.netbirdStatus.NeedsLogin() {
		m.setMessage(warnStyle.Render("  NetBird needs login — run `netbird up` in a terminal"))
		return m, nil
	}
	m.modal = modalConnecting
	m.connectName = netbirdRowName
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		return netbirdDoneMsg{up: true, err: NetBirdUp()}
	})
}
```

- [ ] **Step 3: Route Disconnect — selected NetBird row wins, else active WG**

Replace the `Disconnect` case body:

```go
	case key.Matches(msg, m.keys.Disconnect):
		if m.netbirdSelected() {
			if !m.netbirdStatus.Connected() {
				break
			}
			return m, func() tea.Msg {
				return netbirdDoneMsg{up: false, err: NetBirdDown()}
			}
		}
		if m.activeVPN == "" {
			break
		}
		active := m.activeVPN
		return m, func() tea.Msg {
			err := DisconnectVPN(active)
			return disconnectDoneMsg{err: err}
		}
```

- [ ] **Step 4: Guard Rename and Delete; switch to `selectedConfig()`**

Replace the `Rename` case body:

```go
	case key.Matches(msg, m.keys.Rename):
		if m.netbirdSelected() {
			m.setMessage(warnStyle.Render("  NetBird is managed by its own daemon"))
			break
		}
		selected := m.selectedConfig()
		if selected == "" {
			break
		}
		if selected == m.activeVPN {
			m.setMessage(warnStyle.Render("  Disconnect before renaming"))
			break
		}
		m.modal = modalRenaming
		m.renameOrig = selected
		m.renameInput.SetValue(selected)
		m.renameInput.Focus()
		m.renameInput.CursorEnd()
		return m, textinput.Blink
```

Replace the `Delete` case body:

```go
	case key.Matches(msg, m.keys.Delete):
		if m.netbirdSelected() {
			m.setMessage(warnStyle.Render("  NetBird is managed by its own daemon"))
			break
		}
		selected := m.selectedConfig()
		if selected == "" {
			break
		}
		if selected == m.activeVPN {
			m.setMessage(warnStyle.Render("  Disconnect before deleting"))
			break
		}
		m.modal = modalDeleting
```

In `updateDelete`, replace `name := m.configs[m.cursor]` with:

```go
		name := m.selectedConfig()
		if name == "" {
			m.modal = modalNone
			return m, nil
		}
```

- [ ] **Step 5: Build + run full tests**

Run: `go build . && go test ./...`
Expected: clean build, all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add model.go
git commit -m "feat: route connect/disconnect through NetBird row, guard rename/delete"
```

---

### Task 4: Config panel — render the pinned NetBird row

**Files:**
- Modify: `config_panel.go`

- [ ] **Step 1: Update `renderConfigPanel` list assembly**

Replace the `if len(m.configs) == 0 { ... } else { ... }` block:

```go
	if m.listLen() == 0 {
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("  No configs found."))
		lines = append(lines, dimStyle.Render("  Press "+shortcutKeyStyle.Render("i")+" to import."))
	} else {
		lines = append(lines, "") // top padding
		idx := 0
		if m.netbirdAvail {
			lines = append(lines, m.renderNetbirdItem(idx, width-4))
			idx++
		}
		for _, cfg := range m.configs {
			lines = append(lines, m.renderConfigItem(cfg, idx, width-4))
			idx++
		}
		if len(m.configs) == 0 {
			lines = append(lines, "")
			lines = append(lines, dimStyle.Render("  No configs found."))
			lines = append(lines, dimStyle.Render("  Press "+shortcutKeyStyle.Render("i")+" to import."))
		}
	}
```

- [ ] **Step 2: Make the border reflect either connection**

Replace the border selection:

```go
	border := activeBorderStyle
	if m.activeVPN != "" || m.netbirdStatus.Connected() {
		border = connectedBorderStyle
	}
```

- [ ] **Step 3: Add `renderNetbirdItem`**

Add after `renderConfigItem`:

```go
func (m model) renderNetbirdItem(index, maxWidth int) string {
	isSelected := index == m.cursor

	if isSelected && m.modal == modalConnecting && m.connectName == netbirdRowName {
		return "  " + selectedItemStyle.Render("▸ ") +
			m.spinner.View() + " " +
			selectedItemStyle.Render(netbirdRowName)
	}

	var parts strings.Builder
	if isSelected {
		parts.WriteString("  ")
		parts.WriteString(selectedItemStyle.Render("▸ "))
		parts.WriteString(selectedItemStyle.Render(netbirdRowName))
	} else {
		parts.WriteString("    ")
		parts.WriteString(itemStyle.Render(netbirdRowName))
	}

	if m.netbirdStatus.Connected() {
		parts.WriteString(" ")
		parts.WriteString(connectedIndicator)
	}

	return parts.String()
}
```

- [ ] **Step 4: Build + tests**

Run: `go build . && go test ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add config_panel.go
git commit -m "feat: render pinned NetBird row in config panel"
```

---

### Task 5: Status panel — NetBird detail view

**Files:**
- Modify: `status_panel.go`

- [ ] **Step 1: Branch to the NetBird view when its row is selected**

At the top of `renderStatusPanel`, immediately after the `var lines []string` / `var border lipgloss.Style` declarations, add:

```go
	if m.netbirdSelected() {
		return m.renderNetbirdStatusPanel(width, height)
	}
```

- [ ] **Step 2: Fix config indexing in the preview branch**

Replace the condition `} else if len(m.configs) > 0 && m.cursor < len(m.configs) {` and the line `name := m.configs[m.cursor]` with:

```go
	} else if name := m.selectedConfig(); name != "" {
		// Static preview of highlighted config
		border = inactiveBorderStyle
```

(The rest of the branch already uses `name`.)

- [ ] **Step 3: Add `renderNetbirdStatusPanel`**

Add after `renderStatusPanel`:

```go
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
```

- [ ] **Step 4: Build + tests**

Run: `go build . && go test ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add status_panel.go
git commit -m "feat: NetBird detail view in status panel"
```

---

### Task 6: Title bar + waybar awareness

**Files:**
- Modify: `dashboard.go` (`renderTitleBar`)
- Modify: `main.go` (`printWaybarStatus`)

- [ ] **Step 1: Title bar shows both connections**

Replace the status block in `renderTitleBar`:

```go
	sep := titleAccentStyle.Render("  ─  ")
	var status string
	switch {
	case m.activeVPN != "" && m.netbirdStatus.Connected():
		status = sep + connectedStyle.Render("● "+m.activeVPN) + "  " + connectedStyle.Render("● NetBird")
	case m.activeVPN != "":
		status = sep + connectedStyle.Render("● "+m.activeVPN)
	case m.netbirdStatus.Connected():
		status = sep + connectedStyle.Render("● NetBird")
	default:
		status = sep + dimStyle.Render("○ disconnected")
	}
```

- [ ] **Step 2: Waybar reports connected when either is up**

Replace `printWaybarStatus` in `main.go` (add `"strings"` to imports):

```go
func printWaybarStatus() {
	out := struct {
		Text    string `json:"text"`
		Tooltip string `json:"tooltip"`
		Class   string `json:"class"`
	}{Text: "󰳌", Tooltip: "VPN: Disconnected", Class: "disconnected"}

	var parts []string

	iface := GetActiveVPN()
	if iface != "" {
		if status, err := GetVPNStatus(iface); err == nil {
			parts = append(parts, fmt.Sprintf("VPN: %s\nEndpoint: %s\nTransfer: %s / %s",
				iface, status.Endpoint, status.TransferRx, status.TransferTx))
		}
	}

	if NetBirdAvailable() {
		if nb, err := GetNetBirdStatus(); err == nil && nb.Connected() {
			parts = append(parts, fmt.Sprintf("NetBird: %s\nPeers: %d/%d connected",
				nb.IP, nb.PeersConnected, nb.PeersTotal))
		}
	}

	if len(parts) > 0 {
		out.Text = "󰦝"
		out.Tooltip = strings.Join(parts, "\n")
		out.Class = "connected"
	}

	data, _ := json.Marshal(out)
	fmt.Println(string(data))
}
```

- [ ] **Step 3: Build + tests + vet**

Run: `go build . && go vet ./... && go test ./...`
Expected: clean.

- [ ] **Step 4: Manual smoke test of waybar output**

Run: `go build -o omarchy-vpn . && ./omarchy-vpn --waybar`
Expected: one-line JSON; since netbird is not installed on this machine, identical behavior to before (disconnected or the active WG tunnel).

- [ ] **Step 5: Commit**

```bash
git add dashboard.go main.go
git commit -m "feat: title bar and waybar report NetBird connection"
```

---

### Task 7: Docs + ship

**Files:**
- Modify: `CLAUDE.md` (file table + gotchas)
- Modify: `README.md` (feature mention, if there is a feature list)

- [ ] **Step 1: Update CLAUDE.md**

Add to the architecture file table after the `wireguard.go` row:

```markdown
| `netbird.go` | NetBird backend — `netbird status --json` parsing + up/down (no sudo, talks to daemon) |
```

Add to Gotchas:

```markdown
- NetBird row appears only when the `netbird` binary is installed; it coexists with WG tunnels (overlay mesh, not mutually exclusive)
- `netbird up` is never run when the daemon reports NeedsLogin/SessionExpired — it would block on browser SSO
```

- [ ] **Step 2: Update README.md if it lists features**

Check `README.md`; if it has a feature list, add a line: `- **NetBird aware** — if NetBird is installed, toggle it from the same dashboard (alongside your WireGuard tunnels)`. Match the existing tone.

- [ ] **Step 3: Final verification**

Run: `go build . && go vet ./... && go test ./... -v`
Expected: everything green.

- [ ] **Step 4: Commit, push, PR**

```bash
git add CLAUDE.md README.md
git commit -m "docs: NetBird support"
git push -u origin worktree-feat-netbird-support
gh pr create --title "feat: NetBird support" --body "Closes #22. NetBird appears as a pinned row in the config list (when installed), toggleable via netbird up/down, with status panel + title bar + waybar awareness. Design spec and plan in docs/superpowers/. Coexists with WireGuard tunnels."
```

- [ ] **Step 5: Update plan issue #22 checkboxes**

```bash
gh issue edit 22 -R limehawk/omarchy-vpn --body "<updated body with all boxes checked>"
```
