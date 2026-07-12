package main

import (
	"strings"
	"testing"
)

func TestShortHelpContents(t *testing.T) {
	initColors()
	initStyles()
	h := newHelp()
	h.SetWidth(200)
	view := h.ShortHelpView(newKeyMap().ShortHelp())
	for _, want := range []string{"delete config", "rename config", "•"} {
		if !strings.Contains(view, want) {
			t.Errorf("short help missing %q", want)
		}
	}
	if strings.Contains(view, "│") {
		t.Errorf("short help still contains pipe separator")
	}
}
