package main

import (
	"strings"
	"testing"
)

// A stylesheet shaped like the omarchy default: no VPN rules yet.
const sampleDefaultStyle = `#network {
  margin-right: 13px;
}

#custom-update {
  margin: 0 7.5px;
}
`

func TestStyleWithVPN_FreshInstall_AddsCurrentRules(t *testing.T) {
	out := styleWithVPN(sampleDefaultStyle)

	if !strings.Contains(out, "#custom-vpn {") {
		t.Fatalf("expected base #custom-vpn rule, got:\n%s", out)
	}
	if !strings.Contains(out, "margin-right: 19px;") {
		t.Errorf("expected current 19px margin, got:\n%s", out)
	}
	if !strings.Contains(out, "#custom-vpn.connected {") {
		t.Errorf("expected .connected accent rule, got:\n%s", out)
	}
	if !strings.Contains(out, "#network {") {
		t.Errorf("fresh install dropped existing rules:\n%s", out)
	}
}

// The bug: a pre-fix install wrote margin-right: 13px and no .connected rule,
// and setup refused to touch it ever again. styleWithVPN must overwrite it.
func TestStyleWithVPN_StaleBlock_NormalizesToCurrent(t *testing.T) {
	stale := sampleDefaultStyle + "\n#custom-vpn {\n  margin-right: 13px;\n}\n"

	out := styleWithVPN(stale)

	// Target the VPN rule specifically — #network legitimately uses 13px too.
	if strings.Contains(out, "#custom-vpn {\n  margin-right: 13px;\n}") {
		t.Errorf("stale VPN rule survived; setup did not overwrite:\n%s", out)
	}
	if got := strings.Count(out, "#custom-vpn {"); got != 1 {
		t.Errorf("expected exactly one base rule, got %d:\n%s", got, out)
	}
	if !strings.Contains(out, "margin-right: 19px;") {
		t.Errorf("expected normalized 19px margin:\n%s", out)
	}
	if !strings.Contains(out, "#custom-vpn.connected {") {
		t.Errorf("expected .connected rule to be added on overwrite:\n%s", out)
	}
}

func TestStyleWithVPN_Idempotent(t *testing.T) {
	once := styleWithVPN(sampleDefaultStyle)
	twice := styleWithVPN(once)

	if once != twice {
		t.Errorf("not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
	if got := strings.Count(twice, "#custom-vpn {"); got != 1 {
		t.Errorf("expected exactly one base rule after re-run, got %d", got)
	}
	if got := strings.Count(twice, "#custom-vpn.connected {"); got != 1 {
		t.Errorf("expected exactly one connected rule after re-run, got %d", got)
	}
}

// Pre-#18 installs folded #custom-vpn into the shared selector group. The
// neighbouring rule must survive; only the VPN entry is cleaned out.
func TestStyleWithVPN_LegacySharedGroupInjection_Cleaned(t *testing.T) {
	legacy := `#network {
  margin-right: 13px;
}

#custom-vpn,
#custom-update {
  margin: 0 7.5px;
}

#custom-vpn {
  margin-right: 15px;
}
`
	out := styleWithVPN(legacy)

	if strings.Contains(out, "#custom-vpn,") {
		t.Errorf("legacy shared-group injection survived:\n%s", out)
	}
	if !strings.Contains(out, "#custom-update {") {
		t.Errorf("shared-group neighbour rule was destroyed:\n%s", out)
	}
	if got := strings.Count(out, "#custom-vpn {"); got != 1 {
		t.Errorf("expected exactly one base rule, got %d:\n%s", got, out)
	}
	if strings.Contains(out, "15px") {
		t.Errorf("stale 15px margin survived:\n%s", out)
	}
}

func TestStripVPNStyle_RemovesAllVPNRules(t *testing.T) {
	in := sampleDefaultStyle +
		"\n#custom-vpn {\n  margin-right: 19px;\n}\n#custom-vpn.connected {\n  color: @accent;\n}"

	out := stripVPNStyle(in)

	if strings.Contains(out, "#custom-vpn") {
		t.Errorf("strip left VPN rules behind:\n%s", out)
	}
	if !strings.Contains(out, "#network {") || !strings.Contains(out, "#custom-update {") {
		t.Errorf("strip removed unrelated rules:\n%s", out)
	}
}
