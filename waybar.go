package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const waybarModuleBlock = `  "custom/vpn": {
    "exec": "omarchy-vpn --waybar",
    "return-type": "json",
    "interval": 5,
    "on-click": "omarchy-launch-or-focus-tui omarchy-vpn",
    "tooltip": true
  }`

const waybarCSSConnected = `
#custom-vpn.connected {
  color: @accent;
}`

func waybarConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "waybar", "config.jsonc")
}

func waybarStylePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "waybar", "style.css")
}

func setupWaybar() error {
	if err := patchWaybarConfig(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := patchWaybarStyle(); err != nil {
		return fmt.Errorf("style: %w", err)
	}
	exec.Command("omarchy-restart-waybar").Run()
	fmt.Println("Waybar VPN module installed.")
	return nil
}

func removeWaybar() error {
	if err := unpatchWaybarConfig(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := unpatchWaybarStyle(); err != nil {
		return fmt.Errorf("style: %w", err)
	}
	exec.Command("omarchy-restart-waybar").Run()
	fmt.Println("Waybar VPN module removed.")
	return nil
}

func patchWaybarConfig() error {
	path := waybarConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	if strings.Contains(content, `"custom/vpn"`) {
		return nil
	}

	// Add "custom/vpn" to modules-right before "network"
	content = strings.Replace(content,
		`"network"`,
		"\"custom/vpn\",\n    \"network\"",
		1)

	// Add module definition: find last "}" and insert before it
	// Also add #custom-vpn to the shared CSS selector group
	lastBrace := strings.LastIndex(content, "}")
	if lastBrace == -1 {
		return fmt.Errorf("malformed config")
	}
	// Check if there's a trailing comma situation — find the last non-whitespace before }
	before := strings.TrimRight(content[:lastBrace], " \n\r\t")
	if strings.HasSuffix(before, ",") {
		// Already has trailing comma, just add our block
		content = before + "\n" + waybarModuleBlock + "\n}"
	} else {
		// Add comma after last entry, then our block
		content = before + ",\n" + waybarModuleBlock + "\n}"
	}

	return os.WriteFile(path, []byte(content), 0644)
}

func unpatchWaybarConfig() error {
	path := waybarConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	if !strings.Contains(content, `"custom/vpn"`) {
		return nil
	}

	// Remove from modules-right
	content = strings.Replace(content, "\"custom/vpn\",\n    ", "", 1)

	// Remove module definition block
	start := strings.Index(content, `  "custom/vpn": {`)
	if start != -1 {
		// Find the closing brace of this block
		end := strings.Index(content[start:], "\n  }")
		if end != -1 {
			end += start + 4 // include the "\n  }"
			// Remove trailing comma or newline
			if end < len(content) && content[end] == ',' {
				end++
			}
			if end < len(content) && content[end] == '\n' {
				end++
			}
			content = content[:start] + content[end:]
		}
	}

	return os.WriteFile(path, []byte(content), 0644)
}

func patchWaybarStyle() error {
	path := waybarStylePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	// Add #custom-vpn to the shared selector group with cpu, battery, etc.
	if !strings.Contains(content, "#custom-vpn") {
		content = strings.Replace(content,
			"#custom-update {",
			"#custom-vpn,\n#custom-update {",
			1)
		content += waybarCSSConnected
	}

	return os.WriteFile(path, []byte(content), 0644)
}

func unpatchWaybarStyle() error {
	path := waybarStylePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	if !strings.Contains(content, "#custom-vpn") {
		return nil
	}

	content = strings.Replace(content, "#custom-vpn,\n", "", 1)
	content = strings.Replace(content, waybarCSSConnected, "", 1)

	return os.WriteFile(path, []byte(content), 0644)
}
