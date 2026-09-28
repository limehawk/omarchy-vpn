package main

import (
	"strings"
	"testing"
)

func TestParseAllowedIPs(t *testing.T) {
	prefixes := parseAllowedIPs([]string{"10.0.0.0/24, 192.168.1.0/24", "fd00::/64"})
	if len(prefixes) != 3 {
		t.Fatalf("parseAllowedIPs() returned %d prefixes, want 3", len(prefixes))
	}

	// Bare addresses become single-address prefixes.
	prefixes = parseAllowedIPs([]string{"10.0.0.5"})
	if len(prefixes) != 1 || prefixes[0].Bits() != 32 {
		t.Errorf("bare address = %v, want single /32 prefix", prefixes)
	}

	// Host bits are masked so containment checks work.
	prefixes = parseAllowedIPs([]string{"10.0.0.5/24"})
	if got := prefixes[0].Addr().String(); got != "10.0.0.0" {
		t.Errorf("masked address = %s, want 10.0.0.0", got)
	}

	// Garbage entries are skipped.
	if got := parseAllowedIPs([]string{"not-an-ip, ???"}); len(got) != 0 {
		t.Errorf("garbage input = %v, want empty", got)
	}
}

func TestAllowedIPsOverlap(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{"disjoint subnets", []string{"10.1.0.0/24"}, []string{"10.2.0.0/24"}, false},
		{"identical subnets", []string{"10.1.0.0/24"}, []string{"10.1.0.0/24"}, true},
		{"nested subnets", []string{"10.0.0.0/8"}, []string{"10.1.0.0/24"}, true},
		{"full tunnel vs subnet", []string{"0.0.0.0/0"}, []string{"10.1.0.0/24"}, true},
		{"two full tunnels", []string{"0.0.0.0/0, ::/0"}, []string{"0.0.0.0/0"}, true},
		{"disjoint v4 vs v6", []string{"10.1.0.0/24"}, []string{"fd00::/64"}, false},
		{"comma-separated disjoint", []string{"10.1.0.0/24, 10.2.0.0/24"}, []string{"10.3.0.0/24"}, false},
		{"comma-separated overlapping", []string{"10.1.0.0/24, 10.2.0.0/24"}, []string{"10.2.0.0/24"}, true},
		{"empty side is conservative", nil, []string{"10.1.0.0/24"}, true},
		{"unparseable side is conservative", []string{"garbage"}, []string{"10.1.0.0/24"}, true},
	}
	for _, tt := range tests {
		if got := allowedIPsOverlap(tt.a, tt.b); got != tt.want {
			t.Errorf("%s: allowedIPsOverlap(%v, %v) = %v, want %v", tt.name, tt.a, tt.b, got, tt.want)
		}
	}
}

func TestInterfaceNameError(t *testing.T) {
	if err := interfaceNameError("derby-maryville"); err != nil {
		t.Errorf("15-char name: %v", err)
	}
	if err := interfaceNameError("ok"); err != nil {
		t.Errorf("short name: %v", err)
	}
	if err := interfaceNameError("wg0"); err != nil {
		t.Errorf("wg0: %v", err)
	}
	if err := interfaceNameError("108vpn"); err != nil {
		t.Errorf("108vpn: %v", err)
	}

	if interfaceNameError("-foo") == nil {
		t.Error("leading dash: expected error")
	}
	if got := suggestInterfaceName("-foo"); got != "foo" {
		t.Errorf("suggest -foo: got %q, want foo", got)
	}

	err := interfaceNameError("108")
	if err == nil {
		t.Fatal("numeric name: expected error")
	}
	if !strings.Contains(err.Error(), "number") {
		t.Errorf("numeric name: got %q, want numbers message", err)
	}

	err = interfaceNameError("108-One-Click-VPN-omarchy")
	if err == nil {
		t.Fatal("25-char name: expected error")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("25-char name: got %q, want too-long message", err)
	}

	if err := interfaceNameError(""); err == nil {
		t.Error("empty name: expected error")
	}
	if err := interfaceNameError("has space"); err == nil {
		t.Error("space in name: expected error")
	}
}

func TestSuggestInterfaceName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"derby-maryville", "derby-maryville"},
		{"wg0", "wg0"},
		{"108vpn", "108vpn"},
		{"108", "wg108"},
		{"108-One-Click-VPN-omarchy", "108-One-Click-V"},
		{"123456789012345", "wg1234567890123"},
	}
	for _, tt := range tests {
		if got := suggestInterfaceName(tt.in); got != tt.want {
			t.Errorf("suggestInterfaceName(%q) = %q, want %q", tt.in, got, tt.want)
		}
		if err := interfaceNameError(suggestInterfaceName(tt.in)); err != nil {
			t.Errorf("suggestInterfaceName(%q) still invalid: %v", tt.in, err)
		}
	}
}

func TestConnectVPNRejectsLongName(t *testing.T) {
	err := ConnectVPN("108-One-Click-VPN-omarchy")
	if err == nil {
		t.Fatal("expected error for 25-char name")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("got %q, want too-long message (not wg-quick's 'does not exist')", err)
	}
}

func TestConnectVPNRejectsNumericName(t *testing.T) {
	err := ConnectVPN("108")
	if err == nil {
		t.Fatal("expected error for numeric name")
	}
	if !strings.Contains(err.Error(), "number") {
		t.Errorf("got %q, want numbers message (not resolvectl ifindex error)", err)
	}
}
