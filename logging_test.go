package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogWritesActionsAndRotates(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	old := logger
	defer func() { logger = old }()
	path := logPath()
	os.MkdirAll(filepath.Dir(path), 0o700)
	os.WriteFile(path, make([]byte, maxLogSize+1), 0o600)

	initLog()
	logAction("connect", "derby", nil)
	logAction("connect", "work", errors.New("handshake failed"))

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("oversized log not rotated: %v", err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	for _, want := range []string{"start", "connect", "name=derby", "handshake failed", "ERRO"} {
		if !strings.Contains(got, want) {
			t.Errorf("log missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("log file contains ANSI color codes:\n%s", got)
	}
}
