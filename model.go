package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type modalState int

const (
	modalNone modalState = iota
	modalConnecting
	modalRenaming
	modalDeleting
	modalHelp
	modalImporting
)

type model struct {
	width  int
	height int

	// Config list
	configs []string
	cursor  int

	// Connection state
	activeVPNs []string
	vpnStatus  map[string]VPNStatus

	// NetBird state (only meaningful when netbirdAvail is true)
	netbirdAvail  bool
	netbirdStatus NetBirdStatus

	// Modal state
	modal       modalState
	renameInput textinput.Model
	renameOrig  string
	connectName string
	spinner     spinner.Model
	filePicker  filepicker.Model

	// Help
	keys keyMap
	help help.Model

	// Feedback
	message    string
	messageExp time.Time // when to clear the message
}

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

// isActive reports whether the named config is a currently active tunnel.
func (m model) isActive(name string) bool {
	return name != "" && slices.Contains(m.activeVPNs, name)
}

// refreshVPNState re-queries the active tunnel list and per-tunnel stats.
func (m *model) refreshVPNState() {
	m.activeVPNs = GetActiveVPNs()
	statuses := make(map[string]VPNStatus, len(m.activeVPNs))
	for _, name := range m.activeVPNs {
		if s, err := GetVPNStatus(name); err == nil {
			statuses[name] = s
		}
	}
	m.vpnStatus = statuses
}

// Messages

type connectDoneMsg struct {
	name string
	err  error
}

type disconnectDoneMsg struct {
	name string
	err  error
}

type netbirdDoneMsg struct {
	up  bool // true for `netbird up`, false for `netbird down`
	err error
}

type importDoneMsg struct {
	name string
	err  error
}

type renameDoneMsg struct {
	oldName string
	newName string
	err     error
}

type deleteDoneMsg struct {
	name string
	err  error
}

type statusTickMsg time.Time

func statusTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return statusTickMsg(t)
	})
}

func newFilePicker() filepicker.Model {
	home, _ := os.UserHomeDir()
	fp := filepicker.New()
	fp.CurrentDirectory = home
	fp.AllowedTypes = []string{".conf", ".wg"}
	fp.KeyMap.Back = key.NewBinding(key.WithKeys("h", "backspace", "left"), key.WithHelp("h", "back"))
	return fp
}

func initialModel() model {
	ti := textinput.New()
	ti.Prompt = ""
	s := textinput.DefaultDarkStyles()
	s.Focused.Text = lipgloss.NewStyle().Foreground(green)
	s.Cursor.Color = accent
	ti.SetStyles(s)
	ti.CharLimit = 64

	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(spinnerStyle),
	)

	m := model{
		renameInput: ti,
		spinner:     sp,
		filePicker:  newFilePicker(),
		keys:        newKeyMap(),
		help:        newHelp(),
	}
	m.refreshVPNState()
	m.configs = ListConfigs()
	m.netbirdAvail = NetBirdAvailable()
	if m.netbirdAvail {
		m.netbirdStatus, _ = GetNetBirdStatus()
	}
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(statusTick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		return m, nil

	case statusTickMsg:
		m.refreshVPNState()
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

	case connectDoneMsg:
		m.modal = modalNone
		m.refreshVPNState()
		if msg.err != nil {
			m.setMessage(errorStyle.Render("  " + msg.err.Error()))
		} else {
			m.setMessage(connectedStyle.Render("  Connected to " + msg.name))
		}
		m.configs = ListConfigs()
		return m, nil

	case disconnectDoneMsg:
		m.refreshVPNState()
		if msg.err != nil {
			m.setMessage(errorStyle.Render("  Disconnect failed: " + msg.err.Error()))
		} else {
			m.setMessage(dimStyle.Render("  Disconnected " + msg.name))
		}
		return m, nil

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

	case importDoneMsg:
		if msg.err != nil {
			m.setMessage(errorStyle.Render("  Import failed: " + msg.err.Error()))
		} else {
			m.setMessage(connectedStyle.Render("  Imported " + msg.name))
			m.configs = ListConfigs()
		}
		return m, nil

	case renameDoneMsg:
		m.modal = modalNone
		if msg.err != nil {
			m.setMessage(errorStyle.Render("  Rename failed: " + msg.err.Error()))
		} else {
			m.setMessage(connectedStyle.Render("  Renamed " + msg.oldName + " → " + msg.newName))
			m.configs = ListConfigs()
		}
		return m, nil

	case deleteDoneMsg:
		m.modal = modalNone
		if msg.err != nil {
			m.setMessage(errorStyle.Render("  Delete failed: " + msg.err.Error()))
		} else {
			m.setMessage(dimStyle.Render("  Deleted " + msg.name))
			m.configs = ListConfigs()
			if m.cursor >= len(m.configs) && m.cursor > 0 {
				m.cursor--
			}
		}
		return m, nil

	case spinner.TickMsg:
		if m.modal == modalConnecting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		// Clear message on any keypress
		m.message = ""

		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Help overlay dismisses on any key
		if m.modal == modalHelp {
			m.modal = modalNone
			m.help.ShowAll = false
			return m, nil
		}

		// Route to modal handlers
		switch m.modal {
		case modalRenaming:
			return m.updateRename(msg)
		case modalDeleting:
			return m.updateDelete(msg)
		case modalConnecting:
			return m, nil // block input while connecting
		case modalImporting:
			return m.updateImport(msg)
		}

		// Normal mode keybindings
		return m.updateNormal(msg)
	}

	// Pass non-key messages to filepicker when importing (readDir responses)
	if m.modal == modalImporting {
		return m.updateImportMsg(msg)
	}

	return m, nil
}

func (m *model) updateNormal(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

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

	case key.Matches(msg, m.keys.Connect):
		if m.netbirdSelected() {
			return m.connectNetbird()
		}
		selected := m.selectedConfig()
		if selected == "" {
			break
		}
		if m.isActive(selected) {
			m.setMessage(dimStyle.Render("  Already connected"))
			break
		}
		m.modal = modalConnecting
		m.connectName = selected
		active := slices.Clone(m.activeVPNs)
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			// Only tunnels whose AllowedIPs overlap the new config need to
			// come down first; disjoint tunnels stay connected alongside it.
			for _, name := range conflictingVPNs(selected, active) {
				if err := DisconnectVPN(name); err != nil {
					return connectDoneMsg{name: selected, err: err}
				}
			}
			err := ConnectVPN(selected)
			return connectDoneMsg{name: selected, err: err}
		})

	case key.Matches(msg, m.keys.Disconnect):
		if m.netbirdSelected() {
			if !m.netbirdStatus.Connected() {
				break
			}
			return m, func() tea.Msg {
				return netbirdDoneMsg{up: false, err: NetBirdDown()}
			}
		}
		var name string
		if sel := m.selectedConfig(); m.isActive(sel) {
			name = sel
		} else if len(m.activeVPNs) == 1 {
			name = m.activeVPNs[0]
		}
		if name == "" {
			if len(m.activeVPNs) > 1 {
				m.setMessage(warnStyle.Render("  Select the tunnel to disconnect"))
			}
			break
		}
		return m, func() tea.Msg {
			err := DisconnectVPN(name)
			return disconnectDoneMsg{name: name, err: err}
		}

	case key.Matches(msg, m.keys.Import):
		m.modal = modalImporting
		m.filePicker = newFilePicker()
		m.filePicker, _ = m.filePicker.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.filePicker.Init()

	case key.Matches(msg, m.keys.Rename):
		if m.netbirdSelected() {
			m.setMessage(warnStyle.Render("  NetBird is managed by its own daemon"))
			break
		}
		selected := m.selectedConfig()
		if selected == "" {
			break
		}
		if m.isActive(selected) {
			m.setMessage(warnStyle.Render("  Disconnect before renaming"))
			break
		}
		m.modal = modalRenaming
		m.renameOrig = selected
		m.renameInput.SetValue(selected)
		m.renameInput.Focus()
		m.renameInput.CursorEnd()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Delete):
		if m.netbirdSelected() {
			m.setMessage(warnStyle.Render("  NetBird is managed by its own daemon"))
			break
		}
		selected := m.selectedConfig()
		if selected == "" {
			break
		}
		if m.isActive(selected) {
			m.setMessage(warnStyle.Render("  Disconnect before deleting"))
			break
		}
		m.modal = modalDeleting

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = true
		m.modal = modalHelp
	}

	return m, nil
}

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

func (m *model) updateImport(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.modal = modalNone
		return m, nil
	}

	// Let filepicker handle the key
	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)

	// Check if a file was selected
	if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
		m.modal = modalNone
		return m, m.handleFileSelected(path)
	}

	// Check if user tried to select a disabled file
	if didSelect, _ := m.filePicker.DidSelectDisabledFile(msg); didSelect {
		m.setMessage(errorStyle.Render("  Only .conf and .wg files supported"))
		return m, cmd
	}

	return m, cmd
}

func (m *model) updateImportMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)
	return m, cmd
}

func (m *model) handleFileSelected(path string) tea.Cmd {
	if !ValidateConfig(path) {
		m.setMessage(errorStyle.Render("  Invalid config: missing [Interface]"))
		return nil
	}

	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	name = strings.ReplaceAll(name, " ", "-")
	name = sanitizeName(name)

	return func() tea.Msg {
		err := ImportConfig(path, name)
		return importDoneMsg{name: name, err: err}
	}
}

func (m *model) updateRename(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.modal = modalNone
		m.renameInput.Blur()
		return m, nil

	case "enter":
		newName := sanitizeName(m.renameInput.Value())
		if newName == "" {
			m.setMessage(errorStyle.Render("  Name cannot be empty"))
			return m, nil
		}
		if newName == m.renameOrig {
			m.modal = modalNone
			m.renameInput.Blur()
			return m, nil
		}
		for _, cfg := range m.configs {
			if cfg == newName {
				m.setMessage(errorStyle.Render("  '" + newName + "' already exists"))
				return m, nil
			}
		}
		oldName := m.renameOrig
		m.renameInput.Blur()
		return m, func() tea.Msg {
			err := RenameConfig(oldName, newName)
			return renameDoneMsg{oldName: oldName, newName: newName, err: err}
		}
	}

	// Pass other keys to textinput
	var cmd tea.Cmd
	m.renameInput, cmd = m.renameInput.Update(msg)
	return m, cmd
}

func (m *model) updateDelete(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		name := m.selectedConfig()
		if name == "" {
			m.modal = modalNone
			return m, nil
		}
		m.modal = modalNone
		return m, func() tea.Msg {
			err := RemoveConfig(name)
			return deleteDoneMsg{name: name, err: err}
		}
	default:
		m.modal = modalNone
	}
	return m, nil
}

func (m *model) setMessage(msg string) {
	m.message = msg
	m.messageExp = time.Now().Add(3 * time.Second)
}

func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if result == "" {
		result = "imported"
	}
	return result
}
