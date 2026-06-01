package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const waybarModuleBlock = `  "custom/vpn": {
    "exec": "omarchy-vpn --waybar",
    "return-type": "json",
    "interval": 5,
    "on-click": "omarchy-launch-or-focus-tui omarchy-vpn",
    "tooltip": true
  }`

const waybarCSSBase = `
#custom-vpn {
  margin-right: 19px;
}`

const waybarCSSConnected = `
#custom-vpn.connected {
  color: @accent;
}`

const hyprlandWindowRule = `windowrule = tag +floating-window, match:class org.omarchy.omarchy-vpn`

func waybarConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "waybar", "config.jsonc")
}

func waybarStylePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "waybar", "style.css")
}

func hyprlandConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hypr", "hyprland.conf")
}

func setupWaybar() error {
	if err := patchWaybarConfig(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := patchWaybarStyle(); err != nil {
		return fmt.Errorf("style: %w", err)
	}
	if err := patchHyprlandConfig(); err != nil {
		return fmt.Errorf("hyprland: %w", err)
	}
	fmt.Println("Waybar VPN module installed.")
	reloadWaybar()
	return nil
}

func removeWaybar() error {
	if err := unpatchWaybarConfig(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := unpatchWaybarStyle(); err != nil {
		return fmt.Errorf("style: %w", err)
	}
	if err := unpatchHyprlandConfig(); err != nil {
		return fmt.Errorf("hyprland: %w", err)
	}
	fmt.Println("Waybar VPN module removed.")
	reloadWaybar()
	return nil
}

func reloadWaybar() {
	if path, err := exec.LookPath("omarchy-restart-waybar"); err == nil {
		exec.Command(path).Run()
	}
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

// vpnStyleRule matches a standalone #custom-vpn rule (the base block or the
// .connected variant), along with any blank lines preceding it. It deliberately
// does NOT match the legacy shared selector-group injection
// (#custom-vpn,\n#custom-update { ... }) — that comma form contains a newline
// before the brace, so the neighbouring rule is preserved and cleaned up
// separately by stripVPNStyle.
var vpnStyleRule = regexp.MustCompile(`\n*#custom-vpn[^{}\n]*\{[^}]*\}`)

// stripVPNStyle removes every VPN rule a previous install wrote, regardless of
// which version wrote it (margins have been 13px, 15px, 19px across releases).
func stripVPNStyle(content string) string {
	content = vpnStyleRule.ReplaceAllString(content, "")
	// Pre-#18 installs folded #custom-vpn into the shared selector group.
	content = strings.Replace(content, "#custom-vpn,\n", "", 1)
	return content
}

// styleWithVPN returns the stylesheet with the current VPN rules applied,
// replacing any rules an older install may have written. Version-agnostic and
// idempotent: re-running setup after an upgrade re-normalizes the block instead
// of leaving stale styling in place.
func styleWithVPN(content string) string {
	content = strings.TrimRight(stripVPNStyle(content), "\n")
	return content + waybarCSSBase + waybarCSSConnected
}

func patchWaybarStyle() error {
	path := waybarStylePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(styleWithVPN(string(data))), 0644)
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

	return os.WriteFile(path, []byte(strings.TrimRight(stripVPNStyle(content), "\n")+"\n"), 0644)
}

func patchHyprlandConfig() error {
	path := hyprlandConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	if strings.Contains(content, "org.omarchy.omarchy-vpn") {
		return nil
	}

	content = strings.TrimRight(content, "\n") + "\n\n" + hyprlandWindowRule + "\n"
	return os.WriteFile(path, []byte(content), 0644)
}

func unpatchHyprlandConfig() error {
	path := hyprlandConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	if !strings.Contains(content, "org.omarchy.omarchy-vpn") {
		return nil
	}

	content = strings.Replace(content, "\n"+hyprlandWindowRule, "", 1)
	content = strings.Replace(content, hyprlandWindowRule+"\n", "", 1)

	return os.WriteFile(path, []byte(content), 0644)
}
