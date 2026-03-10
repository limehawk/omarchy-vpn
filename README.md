<p align="center">
  <h1 align="center">omarchy-vpn</h1>
  <p align="center">
    A blazing fast WireGuard VPN manager for your terminal.
    <br />
    Built with <a href="https://github.com/charmbracelet/bubbletea">Bubble Tea v2</a> + <a href="https://github.com/charmbracelet/lipgloss">Lip Gloss v2</a> + <a href="https://github.com/charmbracelet/bubbles">Bubbles v2</a>
  </p>
</p>

<p align="center">
  <a href="#installation"><strong>Install</strong></a> ·
  <a href="#usage"><strong>Usage</strong></a> ·
  <a href="#keybindings"><strong>Keys</strong></a> ·
  <a href="#adding-configs"><strong>Configs</strong></a>
</p>

---

<p align="center">
  <img src="demo.gif" alt="omarchy-vpn demo" width="800" />
</p>

## Features

- **Single-screen dashboard** — two-panel layout, no menus to navigate
- **Live connection stats** — endpoint, transfer, handshake refreshing every second
- **Config preview** — highlight a config to see its details before connecting
- **Inline operations** — rename and delete configs without leaving the dashboard
- **Built-in file picker** — browse and import `.conf` / `.wg` files natively
- **Catppuccin Mocha** — terminal colors with nerd font icons
- **Persistent connections** — quit the TUI, VPN stays connected
- **Waybar integration** — status icon with connection details tooltip, click to launch TUI
- **Zero config** — passwordless via sudoers, just run `omarchy-vpn`

## Installation

### Arch Linux (PKGBUILD)

```bash
git clone https://github.com/limehawk/omarchy-vpn.git
cd omarchy-vpn
makepkg -si
```

`makepkg -si` builds, installs dependencies (`wireguard-tools`, `systemd-resolvconf`), and sets up sudoers — all through pacman.

### From source

```bash
git clone https://github.com/limehawk/omarchy-vpn.git
cd omarchy-vpn
go build -o omarchy-vpn .
sudo install -Dm755 omarchy-vpn /usr/bin/omarchy-vpn
```

You'll need to manually create `/etc/sudoers.d/omarchy-vpn` — see the [PKGBUILD](PKGBUILD) for the required rules.

## Usage

```bash
omarchy-vpn
```

That's the whole interface. Everything happens on one screen.

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |

### Connection

| Key | Action |
|-----|--------|
| `Enter` | Connect to selected config |
| `d` | Disconnect active VPN |

### Config Management

| Key | Action |
|-----|--------|
| `i` | Import config (opens file picker) |
| `r` | Rename selected config (inline) |
| `x` | Delete selected config (with confirmation) |

### General

| Key | Action |
|-----|--------|
| `?` | Toggle help overlay |
| `q` | Quit (VPN stays connected) |
| `Ctrl+C` | Force quit |

## Adding Configs

### Via the app

Press `i` to open the file picker. Navigate to your `.conf` or `.wg` file and select it. The config is validated, sanitized, and copied to `/etc/wireguard/`.

### Manual

```bash
sudo cp *.conf /etc/wireguard/
sudo chmod 600 /etc/wireguard/*.conf
```

## Waybar

The package automatically adds a VPN status module to waybar on install and removes it on uninstall. The icon shows connection state and the tooltip displays endpoint and transfer stats. Click to launch the TUI.

If you need to manually manage the waybar module:

```bash
omarchy-vpn --setup-waybar    # Add module to waybar config + style
omarchy-vpn --remove-waybar   # Remove module from waybar config + style
```

## How It Works

omarchy-vpn is a TUI wrapper around `wg-quick` and `wg show`. Configs live in `/etc/wireguard/` as standard WireGuard `.conf` files. The app manages them with passwordless sudo via a sudoers file installed by the package.

Connect runs `wg-quick up <config>`. Disconnect runs `wg-quick down <config>`. The VPN runs in the kernel — closing the TUI doesn't affect your connection.

## Requirements

- **Go 1.21+** (build only)
- **wireguard-tools** — `wg-quick` and `wg`
- **systemd-resolvconf** — DNS resolution for WireGuard tunnels
- A terminal with nerd font support (for icons)

## Built With

- [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Bubbles v2](https://github.com/charmbracelet/bubbles) — Components (filepicker, help, spinner, textinput)
- [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [Catppuccin Mocha](https://github.com/catppuccin/catppuccin) — Color palette

## License

[MIT](LICENSE)
