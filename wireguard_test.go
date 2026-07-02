package main

import "testing"

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
