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
