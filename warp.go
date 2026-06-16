package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// warpRowName is the display name and connectName sentinel for the pinned
// Cloudflare WARP list row.
const warpRowName = "Cloudflare WARP"

// Connection states reported by `warp-cli status` ("Status update: <State>").
const (
	warpConnected    = "Connected"
	warpDisconnected = "Disconnected"
	warpConnecting   = "Connecting"
	warpNoNetwork    = "No network"
)

// daemonDownMarker appears in warp-cli output when warp-svc is not running.
const daemonDownMarker = "Unable to connect to the CloudflareWARP daemon"

// WarpStatus holds the WARP connection state we display.
//
// We parse the stable *text* output of `warp-cli status` rather than the `-j`
// JSON: the JSON schema is undocumented and shifts between client releases,
// whereas the "Status update: <State>" text line has been stable for years.
type WarpStatus struct {
	State      string // one of the warp* constants, or "" if unparseable
	DaemonDown bool   // warp-svc is not reachable
	Registered bool   // a registration exists (required before connecting)
}

func (s WarpStatus) Connected() bool  { return s.State == warpConnected }
func (s WarpStatus) Connecting() bool { return s.State == warpConnecting }

// NeedsRegistration reports whether `warp-cli connect` would fail (or block on
// Teams SSO) because the client has no registration yet. Only meaningful when
// the daemon is reachable.
func (s WarpStatus) NeedsRegistration() bool {
	return !s.DaemonDown && !s.Registered
}

// WarpAvailable reports whether the warp-cli binary is installed.
func WarpAvailable() bool {
	_, err := exec.LookPath("warp-cli")
	return err == nil
}

// parseWarpStatus extracts connection state from `warp-cli status` output.
// State detection is case-insensitive and ordered so the "connected" substring
// inside "disconnected" never produces a false positive.
func parseWarpStatus(out string, err error) WarpStatus {
	if strings.Contains(out, daemonDownMarker) {
		return WarpStatus{DaemonDown: true}
	}
	low := strings.ToLower(out)
	var s WarpStatus
	switch {
	case strings.Contains(low, "no network"):
		s.State = warpNoNetwork
	case strings.Contains(low, "disconnected"):
		s.State = warpDisconnected
	case strings.Contains(low, "connecting"):
		s.State = warpConnecting
	case strings.Contains(low, "connected"):
		s.State = warpConnected
	}
	// Unparseable output plus a non-zero exit means the daemon is unreachable;
	// don't invent a state.
	if s.State == "" && err != nil {
		s.DaemonDown = true
	}
	return s
}

// GetWarpStatus queries warp-cli for the current connection state, and (when
// not connected) whether the client is registered. A zero-value status with
// DaemonDown set means warp-svc is unreachable.
func GetWarpStatus() WarpStatus {
	out, err := exec.Command("warp-cli", "status").CombinedOutput()
	s := parseWarpStatus(string(out), err)
	if s.DaemonDown {
		return s
	}
	if s.Connected() {
		s.Registered = true
	} else {
		s.Registered = warpRegistered()
	}
	return s
}

// warpRegistered reports whether a WARP registration exists. `registration
// show` exits non-zero when the client has never been registered.
func warpRegistered() bool {
	return exec.Command("warp-cli", "registration", "show").Run() == nil
}

func WarpUp() error {
	out, err := exec.Command("warp-cli", "connect").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", extractError(string(out), err))
	}
	return nil
}

func WarpDown() error {
	out, err := exec.Command("warp-cli", "disconnect").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", extractError(string(out), err))
	}
	return nil
}
