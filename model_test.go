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
