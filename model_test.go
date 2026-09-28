package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

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

// TestCursorHelpersBothPinned exercises the two-pinned-row layout
// [NetBird, WARP] + configs, which is the highest-risk index math.
func TestCursorHelpersBothPinned(t *testing.T) {
	m := model{netbirdAvail: true, warpAvail: true, configs: []string{"alpha", "beta"}}

	if got := m.listLen(); got != 4 {
		t.Errorf("listLen() = %d, want 4", got)
	}
	if got := m.pinnedCount(); got != 2 {
		t.Errorf("pinnedCount() = %d, want 2", got)
	}

	cases := []struct {
		cursor      int
		wantNetbird bool
		wantWarp    bool
		wantConfig  string
	}{
		{0, true, false, ""},
		{1, false, true, ""},
		{2, false, false, "alpha"},
		{3, false, false, "beta"},
	}
	for _, c := range cases {
		m.cursor = c.cursor
		if got := m.netbirdSelected(); got != c.wantNetbird {
			t.Errorf("cursor %d: netbirdSelected() = %v, want %v", c.cursor, got, c.wantNetbird)
		}
		if got := m.warpSelected(); got != c.wantWarp {
			t.Errorf("cursor %d: warpSelected() = %v, want %v", c.cursor, got, c.wantWarp)
		}
		if got := m.selectedConfig(); got != c.wantConfig {
			t.Errorf("cursor %d: selectedConfig() = %q, want %q", c.cursor, got, c.wantConfig)
		}
	}
}

// TestCursorHelpersWarpOnly: WARP available, NetBird absent — WARP takes row 0.
func TestCursorHelpersWarpOnly(t *testing.T) {
	m := model{warpAvail: true, configs: []string{"alpha"}}

	if got := m.listLen(); got != 2 {
		t.Errorf("listLen() = %d, want 2", got)
	}
	m.cursor = 0
	if !m.warpSelected() {
		t.Error("cursor 0 should select WARP row when NetBird is absent")
	}
	if m.netbirdSelected() {
		t.Error("netbirdSelected() must be false when NetBird unavailable")
	}
	if got := m.selectedConfig(); got != "" {
		t.Errorf("selectedConfig() on WARP row = %q, want \"\"", got)
	}
	m.cursor = 1
	if m.warpSelected() {
		t.Error("cursor 1 should not select WARP row")
	}
	if got := m.selectedConfig(); got != "alpha" {
		t.Errorf("selectedConfig() = %q, want alpha", got)
	}
}

func TestRestoreSelectionKeepsConfigWhenWarpDisappears(t *testing.T) {
	m := model{netbirdAvail: true, warpAvail: true, configs: []string{"alpha", "beta"}, cursor: 2}
	if m.selectedConfig() != "alpha" {
		t.Fatalf("precondition: cursor 2 should be alpha, got %q", m.selectedConfig())
	}
	name := m.selectedRowName()
	m.warpAvail = false
	m.restoreSelection(name)
	if got := m.selectedConfig(); got != "alpha" {
		t.Errorf("after WARP hid: selectedConfig() = %q, want alpha", got)
	}
}

func TestImportTooLongPromptsRename(t *testing.T) {
	initColors()
	path := writeTempConf(t, "108-One-Click-VPN-omarchy.conf")
	m := testModel()
	m.configs = []string{"derby-maryville"}

	cmd := m.handleFileSelected(path)
	if m.modal != modalRenaming {
		t.Fatalf("modal = %v, want renaming", m.modal)
	}
	if m.importPath != path {
		t.Fatalf("importPath = %q, want %q", m.importPath, path)
	}
	if got := m.renameInput.Value(); got != "108-One-Click-V" {
		t.Errorf("rename input = %q, want suggested valid name", got)
	}
	if cmd == nil {
		t.Error("expected blink cmd so the input is focused")
	}
}

func TestImportShortNameDoesNotPromptRename(t *testing.T) {
	initColors()
	path := writeTempConf(t, "office.conf")
	m := testModel()

	_ = m.handleFileSelected(path)
	if m.modal == modalRenaming {
		t.Fatal("short name should import, not prompt rename")
	}
	if m.importPath != "" {
		t.Fatalf("importPath = %q, want empty", m.importPath)
	}
}

func TestImportRenameEscCancels(t *testing.T) {
	initColors()
	m := testModel()
	m.modal = modalRenaming
	m.importPath = "/tmp/foo.conf"
	m.renameInput.SetValue("108-One-Click-VPN-omarchy")

	_, cmd := m.updateRename(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.modal != modalNone {
		t.Errorf("modal = %v, want none", m.modal)
	}
	if m.importPath != "" {
		t.Errorf("importPath = %q, want empty", m.importPath)
	}
	if cmd != nil {
		t.Error("esc must not import")
	}
}

func TestImportNumericNamePromptsRename(t *testing.T) {
	initColors()
	path := writeTempConf(t, "108.conf")
	m := testModel()

	_ = m.handleFileSelected(path)
	if m.modal != modalRenaming {
		t.Fatalf("modal = %v, want renaming", m.modal)
	}
	if m.importPath != path {
		t.Fatalf("importPath = %q, want %q", m.importPath, path)
	}
	if got := m.renameInput.Value(); got != "wg108" {
		t.Errorf("rename input = %q, want wg108", got)
	}
}

func TestImportRenameRejectsNumeric(t *testing.T) {
	initColors()
	m := testModel()
	m.modal = modalRenaming
	m.importPath = "/tmp/foo.conf"
	m.renameInput.SetValue("108")

	_, cmd := m.updateRename(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.modal != modalRenaming {
		t.Errorf("modal = %v, want renaming", m.modal)
	}
	if cmd != nil {
		t.Error("must not import a numeric name")
	}
	if !strings.Contains(m.message, "number") {
		t.Errorf("message = %q, want numbers hint", m.message)
	}
}

func TestImportRenameRejectsStillTooLong(t *testing.T) {
	initColors()
	m := testModel()
	m.modal = modalRenaming
	m.importPath = "/tmp/foo.conf"
	m.renameInput.SetValue("still-way-too-long")

	_, cmd := m.updateRename(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.modal != modalRenaming {
		t.Errorf("modal = %v, want renaming", m.modal)
	}
	if m.importPath == "" {
		t.Error("importPath cleared; should keep pending import")
	}
	if cmd != nil {
		t.Error("must not import a too-long name")
	}
}

func TestImportRenameAcceptsShortName(t *testing.T) {
	initColors()
	m := testModel()
	m.modal = modalRenaming
	m.importPath = "/tmp/foo.conf"
	m.renameInput.SetValue("one-click")

	_, cmd := m.updateRename(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.modal != modalNone {
		t.Errorf("modal = %v, want none", m.modal)
	}
	if m.importPath != "" {
		t.Errorf("importPath = %q, want empty", m.importPath)
	}
	if cmd == nil {
		t.Fatal("expected ImportConfig cmd")
	}
}

func TestConnectNumericNamePromptsRename(t *testing.T) {
	initColors()
	m := testModel()
	m.configs = []string{"108"}
	m.cursor = 0

	_, cmd := m.toggleSelected()
	if m.modal != modalRenaming {
		t.Fatalf("modal = %v, want renaming", m.modal)
	}
	if got := m.renameOrig; got != "108" {
		t.Errorf("renameOrig = %q", got)
	}
	if got := m.renameInput.Value(); got != "wg108" {
		t.Errorf("rename input = %q, want wg108", got)
	}
	if cmd == nil {
		t.Error("expected blink cmd")
	}
}

func TestConnectLongNamePromptsRename(t *testing.T) {
	initColors()
	m := testModel()
	m.configs = []string{"108-One-Click-VPN-omarchy"}
	m.cursor = 0

	_, cmd := m.toggleSelected()
	if m.modal != modalRenaming {
		t.Fatalf("modal = %v, want renaming", m.modal)
	}
	if m.importPath != "" {
		t.Error("existing config rename must not set importPath")
	}
	if got := m.renameOrig; got != "108-One-Click-VPN-omarchy" {
		t.Errorf("renameOrig = %q", got)
	}
	if cmd == nil {
		t.Error("expected blink cmd")
	}
}

func TestConfigPanelShowsImportRenamePrompt(t *testing.T) {
	initColors()
	m := testModel()
	m.modal = modalRenaming
	m.importPath = "/tmp/foo.conf"
	m.renameInput.SetValue("108-One-Click-V")
	m.renameInput.Focus()

	got := m.renderConfigPanel(36, 16)
	if !strings.Contains(got, "108-One-Click-V") {
		t.Fatalf("missing rename input:\n%s", got)
	}
	if !strings.Contains(got, "max 15") {
		t.Fatalf("missing length hint:\n%s", got)
	}
}

func testModel() model {
	return model{renameInput: textinput.New()}
}

func writeTempConf(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("[Interface]\nPrivateKey = dGVzdA==\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportWithHooksAsksFirst(t *testing.T) {
	initColors()
	path := filepath.Join(t.TempDir(), "lan.conf")
	conf := "[Interface]\nPrivateKey = dGVzdA==\nTable = off\n  postup = ip route add 192.168.1.0/24 dev %i\nPreDown = ip route del 192.168.1.0/24 dev %i\n"
	if err := os.WriteFile(path, []byte(conf), 0600); err != nil {
		t.Fatal(err)
	}
	m := testModel()
	m.width, m.height = 120, 40

	if cmd := m.handleFileSelected(path); cmd != nil {
		t.Fatal("hook config must not import before confirmation")
	}
	if m.modal != modalHookConfirm || m.importName != "lan" || len(m.importHooks) != 2 {
		t.Fatalf("modal=%v name=%q hooks=%q, want confirm for lan with 2 hooks", m.modal, m.importName, m.importHooks)
	}
	if !strings.Contains(m.View().Content, "ip route add 192.168.1.0/24") {
		t.Error("overlay must show the exact hook command")
	}

	_, cmd := m.updateHookConfirm(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if cmd != nil || m.modal != modalNone || m.importPath != "" {
		t.Errorf("declining must cancel cleanly: cmd=%v modal=%v path=%q", cmd != nil, m.modal, m.importPath)
	}

	_ = m.handleFileSelected(path)
	if _, cmd := m.updateHookConfirm(tea.KeyPressMsg{Code: 'y', Text: "y"}); cmd == nil {
		t.Error("confirming must return the authenticated install cmd")
	}
}

func TestImportRefusesExistingName(t *testing.T) {
	initColors()
	path := writeTempConf(t, "office.conf")
	m := testModel()
	m.configs = []string{"office"}

	if cmd := m.handleFileSelected(path); cmd != nil {
		t.Fatal("import over an existing config must not run")
	}
	if !strings.Contains(m.message, "already exists") {
		t.Errorf("message = %q, want already-exists warning", m.message)
	}
}
