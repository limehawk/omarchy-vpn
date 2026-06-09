# NetBird Support — Design

**Date:** 2026-06-09
**Status:** Approved

## Goal

Promote NetBird from "ignored foreign WireGuard interface" to a first-class
citizen in omarchy-vpn: visible, connectable, and disconnectable from the TUI
and reflected in the waybar module.

## Background

NetBird is a WireGuard-based mesh VPN managed by its own daemon
(`netbird.service`). The `netbird` CLI talks to the daemon over a unix socket
(world-writable by default), so **no sudo is required** — unlike the
`wg-quick` path. Commit f238bad made `GetActiveVPN()` deliberately skip
NetBird's `wt0` interface; this feature builds on that by giving NetBird its
own dedicated state instead.

Verified against netbird v0.72.1 (current AUR version):

- `netbird up` / `netbird down` connect/disconnect.
- `netbird status --json` returns `daemonStatus` (`Idle`, `Connecting`,
  `Connected`, `NeedsLogin`, `LoginFailed`, `SessionExpired`), `netbirdIp`,
  `fqdn`, `management{url,connected}`, `peers{total,connected,details[]}`
  with per-peer `transferReceived`/`transferSent`.

## Design

### Config list (left panel)

- When `exec.LookPath("netbird")` succeeds, a pinned **NetBird** row appears
  at the **top** of the config list, above WireGuard configs.
- If NetBird is not installed the row is absent and nothing changes for
  existing users.
- The row gets the same `●` connected indicator as configs.
- Enter/space on the row connects (`netbird up`); `d` disconnects
  (`netbird down`).
- Rename (`r`), delete (`x`) on the row flash "NetBird is managed by its own
  daemon" and do nothing.
- Cursor math: the visible list is `[NetBird?] + configs`. Helper methods on
  the model translate cursor index → config to avoid off-by-ones.

### Coexistence

NetBird is a mesh overlay, not a full tunnel. It coexists with WireGuard
tunnels:

- Toggling a WG config never touches NetBird and vice versa.
- `activeVPN` remains WG-only. A new `netbirdStatus` model field tracks
  NetBird, refreshed on the existing 1-second `statusTick` by parsing
  `netbird status --json`.

### Status panel (right panel)

- Cursor on the NetBird row → NetBird details: status badge, NetBird IP,
  FQDN, peers connected/total, management URL, summed transfer.
- Cursor on a config row → existing behavior, unchanged.

### Login / daemon edge cases

- If `daemonStatus` is `NeedsLogin`, `LoginFailed`, or `SessionExpired`,
  do **not** run `netbird up` (it would block on a browser SSO flow).
  Flash: "NetBird needs login — run `netbird up` in a terminal."
- Daemon not running → `netbird status` errors; the row still shows, status
  panel says so, and toggle errors surface as flash messages.

### Title bar & waybar

- Title bar shows both connections when both are up
  (e.g. `● homelab ● NetBird`).
- `--waybar` reports class `connected` if **either** WG or NetBird is up;
  tooltip lists both.

## Components

| File | Change |
|------|--------|
| `netbird.go` (new) | `NetBirdAvailable()`, `NetBirdStatus` struct + `GetNetBirdStatus()` (parses `status --json`), `NetBirdUp()`, `NetBirdDown()` |
| `model.go` | `netbirdAvail`/`netbirdStatus` fields, cursor helpers, connect/disconnect routing, tick refresh, new `netbirdDoneMsg` |
| `config_panel.go` | Render pinned NetBird row |
| `status_panel.go` | NetBird detail view when row selected |
| `dashboard.go` | Title bar dual-connection state |
| `waybar.go` / `main.go` | `--waybar` NetBird awareness |

## Testing

- Unit tests: `netbird status --json` parsing from a fixture; cursor offset
  helpers with and without the NetBird row.
- Manual: TUI behavior with netbird absent (row hidden, everything as
  before) and with the binary present.

## Out of scope

- NetBird login flow / SSO from inside the TUI.
- NetBird peer management, routes, or settings.
- Tailscale or other mesh VPNs (same pattern could be reused later).
