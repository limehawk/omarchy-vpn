package main

import (
	"encoding/json"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
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

	iface := GetActiveVPN()
	if iface != "" {
		status, err := GetVPNStatus(iface)
		if err == nil {
			out.Text = "󰦝"
			out.Tooltip = fmt.Sprintf("VPN: %s\nEndpoint: %s\nTransfer: %s / %s",
				iface, status.Endpoint, status.TransferRx, status.TransferTx)
			out.Class = "connected"
		}
	}

	data, _ := json.Marshal(out)
	fmt.Println(string(data))
}
