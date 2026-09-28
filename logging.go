package main

import (
	"io"
	"os"
	"path/filepath"

	"charm.land/log/v2"
)

// maxLogSize rotates the log to omarchy-vpn.log.1 at startup once it grows
// past this, so it can never fill the disk.
const maxLogSize = 1 << 20

// logger discards until initLog runs, so --waybar, --setup and tests never
// write a log or print into the TUI.
var logger = log.New(io.Discard)

// logPath is $XDG_STATE_HOME/omarchy-vpn/omarchy-vpn.log
// (~/.local/state/omarchy-vpn/omarchy-vpn.log by default).
func logPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "omarchy-vpn", "omarchy-vpn.log")
}

// initLog opens the log file for the TUI. Failure is silent: logging must
// never stop the app from starting.
func initLog() {
	path := logPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	if fi, err := os.Stat(path); err == nil && fi.Size() > maxLogSize {
		os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	logger = log.NewWithOptions(f, log.Options{ReportTimestamp: true, TimeFormat: "2006-01-02 15:04:05"})
	logger.Info("start", "version", displayVersion())
}

// logAction records one user action and its outcome. Only names and error
// text are logged, never config contents or keys.
func logAction(action, name string, err error) {
	if err != nil {
		logger.Error(action, "name", name, "err", err)
		return
	}
	logger.Info(action, "name", name)
}
