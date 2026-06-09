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
