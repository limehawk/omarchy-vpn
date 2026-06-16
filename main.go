package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// version is the build version, injected via -ldflags "-X main.version=$pkgver".
// Plain `go build` leaves it as "dev".
var version = "dev"

// displayVersion renders the version for humans: "v0.1.0" for releases, "dev"
// for un-stamped builds.
func displayVersion() string {
	if version == "dev" {
		return "dev"
	}
	return "v" + version
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("omarchy-vpn " + displayVersion())
			return
		case "--waybar":
			printWaybarStatus()
			return
		case "--setup-waybar":
			if err := setupWaybar(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "--remove-waybar":
			if err := removeWaybar(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	initColors()
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printWaybarStatus() {
	out := struct {
		Text    string `json:"text"`
		Tooltip string `json:"tooltip"`
		Class   string `json:"class"`
	}{Text: "󰳌", Tooltip: "VPN: Disconnected", Class: "disconnected"}

	var parts []string

	for _, iface := range GetActiveVPNs() {
		if status, err := GetVPNStatus(iface); err == nil {
			parts = append(parts, fmt.Sprintf("VPN: %s\nEndpoint: %s\nTransfer: %s / %s",
				iface, status.Endpoint, status.TransferRx, status.TransferTx))
		}
	}

	if NetBirdAvailable() {
		if nb, err := GetNetBirdStatus(); err == nil && nb.Connected() {
			parts = append(parts, fmt.Sprintf("NetBird: %s\nPeers: %d/%d connected",
				nb.IP, nb.PeersConnected, nb.PeersTotal))
		}
	}

	if WarpAvailable() {
		if GetWarpStatus().Connected() {
			parts = append(parts, "Cloudflare WARP: Connected")
		}
	}

	if len(parts) > 0 {
		out.Text = "󰦝"
		out.Tooltip = strings.Join(parts, "\n")
		out.Class = "connected"
	}

	data, _ := json.Marshal(out)
	fmt.Println(string(data))
}
