# omarchy-vpn

WireGuard VPN manager TUI for Omarchy (Arch Linux + Hyprland). Single-screen dashboard built with Bubble Tea v2.

## Build & Install

```bash
makepkg -si          # Build package and install via pacman (installs deps + sudoers)
makepkg -fs          # Rebuild after code changes (-f forces, -s resolves deps)
go build -o omarchy-vpn .  # Build binary only (no package/sudoers)
```

## Stack

**Always use Charm ecosystem components.** No external TUI tools (gum, fzf, etc.) — everything must be a native Bubble Tea component rendered inline.

| Package | Version | Import |
|---------|---------|--------|
| Bubble Tea | v2 | `charm.land/bubbletea/v2` |
| Bubbles | v2 | `charm.land/bubbles/v2/*` |
| Lip Gloss | v2 | `charm.land/lipgloss/v2` |

**Components in use:** `filepicker`, `help`, `key`, `spinner`, `textinput`

When adding new features, check bubbles v2 for an existing component first (viewport, table, list, progress, textarea, etc.) before building custom UI.

## Architecture

Single Bubble Tea program with modal states instead of view routing. All UI is one `View()` function composing panels.

| File | Role |
|------|------|
| `main.go` | Entry point — `tea.NewProgram()` (alt screen set declaratively in View) |
| `model.go` | Model struct, all message types, `Update()` with modal dispatch, filepicker |
| `dashboard.go` | `View()` — title bar + panels + help bar, returns `tea.View` |
| `config_panel.go` | Left panel — config list with inline rename/delete/connecting states |
| `status_panel.go` | Right panel — live stats (connected) or config preview (disconnected) |
| `wireguard.go` | Backend — config name validation + all `sudo` exec calls to wg-quick/wg |
| `waybar.go` | Waybar module — `--waybar` JSON output + `--setup`/`--remove` config management |
| `styles.go` | Semantic color variables + all lipgloss styles (initialized by `initStyles()`) |
| `theme.go` | ANSI terminal color assignments → `initColors()` → `initStyles()` |
| `help.go` | `keyMap` with `key.Binding` definitions, `help.Model` with custom styles |

## Key Patterns

- **Declarative View** — `View()` returns `tea.View` with `v.AltScreen = true` (v2 pattern, no `tea.WithAltScreen()`)
- **`tea.KeyPressMsg`** not `tea.KeyMsg` (v2 split key events into press/release)
- **Modal states** (`modalNone`, `modalConnecting`, `modalRenaming`, `modalDeleting`, `modalHelp`, `modalImporting`) replace view routing
- **Inline operations** — rename shows `textinput` in the list item, delete shows `[y/n]` confirmation
- **Inline filepicker** — import uses `bubbles/filepicker` rendered fullscreen, filtered to `.conf` only
- **Flash messages** — `setMessage()` sets 3-second expiry, replaces help bar
- **Bubbles help component** — `help.Model` generates both the bottom bar (`ShortHelp`) and the help overlay (`FullHelp`) from `keyMap`
- **`extractError()`** strips `[#]` trace lines from wg-quick output, returns only the meaningful error
- **`ParseConfigFile()`** uses `sudo cat` to read root-owned configs, falls back to `os.ReadFile()`

## Gotchas

- All WireGuard operations need passwordless sudo — PKGBUILD installs sudoers rules for `%wheel`
- `systemd-resolvconf` is required (provides `resolvconf` shim for wg-quick DNS)
- Config names are sanitized to `[a-zA-Z0-9_-]` only
- Cannot rename or delete the active VPN — must disconnect first

## AUR Publishing

The package is published to AUR at `https://aur.archlinux.org/packages/omarchy-vpn`. AUR is a **separate git server** from GitHub — two independent repos.

| Repo | URL | Contents |
|------|-----|----------|
| GitHub | `github.com/limehawk/omarchy-vpn` | Source code + local-build PKGBUILD |
| AUR | `aur.archlinux.org/omarchy-vpn.git` | Tarball PKGBUILD + `.SRCINFO` + `.install` only |

**AUR SSH:** Requires an SSH key registered with AUR. See [AUR submission guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines).

**Release process:**

```bash
# 1. Tag + release on GitHub
git tag v0.X.X && git push && git push --tags
gh release create v0.X.X

# 2. Get new tarball checksum
curl -sL "https://github.com/limehawk/omarchy-vpn/archive/v0.X.X.tar.gz" | sha256sum

# 3. Clone AUR repo (if not already cloned — /tmp is fine, it's throwaway)
git clone ssh://aur@aur.archlinux.org/omarchy-vpn.git /tmp/omarchy-vpn-aur

# 4. Update PKGBUILD: bump pkgver + sha256sums
# 5. Regenerate .SRCINFO
cd /tmp/omarchy-vpn-aur
makepkg --printsrcinfo > .SRCINFO

# 6. Commit + push to AUR (branch MUST be master, not main)
git add PKGBUILD .SRCINFO
git commit -m "Update to 0.X.X: description"
git push
```


## Theming

Uses ANSI terminal colors (0-8) via `lipgloss.ANSIColor()`. The terminal emulator maps these to whatever palette the active system theme defines, so colors update automatically when the theme changes. No theme files are read directly.

**Do NOT hardcode hex colors.** All colors come from `theme.go` → `initColors()` → `initStyles()`.
