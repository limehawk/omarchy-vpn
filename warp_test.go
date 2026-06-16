package main

import (
	"errors"
	"testing"
)

func TestParseWarpStatus(t *testing.T) {
	cases := []struct {
		name       string
		out        string
		err        error
		wantState  string
		wantDaemon bool
	}{
		{"connected", "Status update: Connected", nil, warpConnected, false},
		{"disconnected", "Status update: Disconnected\nReason: Manual Disconnection", nil, warpDisconnected, false},
		{"connecting", "Status update: Connecting", nil, warpConnecting, false},
		{"no network", "Status update: No network", nil, warpNoNetwork, false},
		{"bare connected", "Connected", nil, warpConnected, false},
		{"daemon down", daemonDownMarker + ": No such file or directory (os error 2)", errors.New("exit 1"), "", true},
		{"unparseable with error", "weird output", errors.New("exit 1"), "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := parseWarpStatus(c.out, c.err)
			if s.State != c.wantState {
				t.Errorf("State = %q, want %q", s.State, c.wantState)
			}
			if s.DaemonDown != c.wantDaemon {
				t.Errorf("DaemonDown = %v, want %v", s.DaemonDown, c.wantDaemon)
			}
		})
	}
}

func TestWarpStatusPredicates(t *testing.T) {
	// "disconnected" must not be mistaken for "connected".
	if parseWarpStatus("Status update: Disconnected", nil).Connected() {
		t.Error("Disconnected should not report Connected()")
	}
	if !parseWarpStatus("Status update: Connected", nil).Connected() {
		t.Error("Connected should report Connected()")
	}
}

func TestWarpNeedsRegistration(t *testing.T) {
	// Daemon down is never "needs registration".
	if (WarpStatus{DaemonDown: true}).NeedsRegistration() {
		t.Error("daemon down should not report NeedsRegistration()")
	}
	// Reachable + unregistered.
	if !(WarpStatus{State: warpDisconnected, Registered: false}).NeedsRegistration() {
		t.Error("disconnected + unregistered should report NeedsRegistration()")
	}
	// Reachable + registered.
	if (WarpStatus{State: warpDisconnected, Registered: true}).NeedsRegistration() {
		t.Error("registered should not report NeedsRegistration()")
	}
}
