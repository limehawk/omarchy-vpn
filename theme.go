package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
)

type themeColors struct {
	mainFg     string
	mainBg     string
	title      string
	hiFg       string
	selectedBg string
	selectedFg string
	inactiveFg string
	procMisc   string
	divLine    string
}

// loadBtopTheme reads ~/.config/omarchy/current/theme/btop.theme
func loadBtopTheme() *themeColors {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	path := filepath.Join(home, ".config", "omarchy", "current", "theme", "btop.theme")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	t := &themeColors{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "theme[") {
			continue
		}

		// theme[key]="value"
		bracket := strings.Index(line, "]")
		if bracket == -1 {
			continue
		}
		key := line[6:bracket]
		val := line[bracket+1:]
		val = strings.TrimPrefix(val, "=")
		val = strings.Trim(val, "\"")

		switch key {
		case "main_fg":
			t.mainFg = val
		case "main_bg":
			t.mainBg = val
		case "title":
			t.title = val
		case "hi_fg":
			t.hiFg = val
		case "selected_bg":
			t.selectedBg = val
		case "selected_fg":
			t.selectedFg = val
		case "inactive_fg":
			t.inactiveFg = val
		case "proc_misc":
			t.procMisc = val
		case "div_line":
			t.divLine = val
		}
	}

	if t.mainFg == "" || t.mainBg == "" {
		return nil
	}
	return t
}

func initColors() {
	t := loadBtopTheme()
	if t == nil {
		// Fallback: ANSI colors for non-omarchy systems
		green = lipgloss.ANSIColor(2)
		red = lipgloss.ANSIColor(1)
		yellow = lipgloss.ANSIColor(3)
		accent = lipgloss.ANSIColor(4)
		textCol = lipgloss.ANSIColor(7)
		dimCol = lipgloss.ANSIColor(8)
		borderCol = lipgloss.ANSIColor(7)
		base = lipgloss.ANSIColor(0)
	} else {
		green = lipgloss.Color(t.hiFg)
		red = lipgloss.Color(t.procMisc)
		yellow = lipgloss.Color(t.inactiveFg)
		accent = lipgloss.Color(t.hiFg)
		textCol = lipgloss.Color(t.mainFg)
		dimCol = lipgloss.Color(t.inactiveFg)
		borderCol = lipgloss.Color(t.divLine)
		base = lipgloss.Color(t.mainBg)
	}

	initStyles()
}
