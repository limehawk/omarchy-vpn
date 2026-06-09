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
