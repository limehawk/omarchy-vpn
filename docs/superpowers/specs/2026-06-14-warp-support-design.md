# Cloudflare WARP Support — Design

**Date:** 2026-06-14
**Status:** Approved

## Goal

Give Cloudflare WARP the same first-class treatment NetBird got: a pinned row
in the config list that connects/disconnects via the terminal `warp-cli`, a
status detail panel, and title-bar + waybar reporting — all without the WARP
GUI client. WARP becomes a second daemon-managed VPN row alongside NetBird.

This is **active control, not just status display.** The row drives
`warp-cli connect` / `warp-cli disconnect`. It degrades to awareness-only in
exactly two guard states (needs-registration, daemon-down), where it shows the
state but will not auto-connect because connecting would block on browser SSO
or simply fail.

## Background

WARP is Cloudflare's device VPN client. The original request said
"cloudflared," but cloudflared (Tunnel/Access) is a different product;
the intended capability is **WARP via `warp-cli`** — connect *this device* to
Cloudflare's network / a Zero Trust org, the same role WireGuard and NetBird
play here.

Like NetBird, `warp-cli` talks to its daemon (`warp-svc`, a systemd service)
over a local socket, so **no sudo is expected** for connect/disconnect/status.
This mirrors the NetBird path, not the `wg-quick` path.

Verified against `warp-cli 2026.4.1390.0` (installed on the dev machine):

- Subcommands include `connect`, `disconnect`, `status`, `stats`,
  `registration`, `mode`, `settings`.
- A global `-j/--json` flag pretty-prints output as JSON
  (e.g. `warp-cli -j status`).
- With the daemon stopped, JSON output is
  `{"code":"FailedToConnectToDaemon","error":"..."}` — a `code` field is the
  daemon-down signal.

**Pinned at implementation (not fabricated here):** the daemon was not running
on the dev machine and cannot be started without sudo, so the exact JSON field
names for a *connected* `warp-cli -j status` / `warp-cli -j stats` payload
(connection state, mode, account type, endpoint, transfer counters) will be
confirmed from a live capture or Cloudflare docs during implementation.
Parsing stays **defensive** — render only fields that are present — exactly as
NetBird does. The semantic states the parser must distinguish are fixed
(below); only the literal key names are deferred.

## Design

### Two pinned rows (the one new architectural decision)

Today the model hardcodes a *single* pinned row at `cursor == 0`
(`netbirdSelected()` is `cursor == 0`; `selectedConfig()` does one `i--`).
Adding WARP makes that assumption wrong. **Approach B (approved):** explicit
availability booleans plus thin index helpers, NOT a generic pinned-backend
interface (premature for two similar-but-distinct backends).

- Fixed row order: **NetBird first, WARP second**, then WireGuard configs.
  Visible list is `[NetBird?] + [WARP?] + configs`. Keeping NetBird at index 0
  leaves the existing NetBird path and its tests almost untouched.
- New helpers centralize the cursor math in one tested place:
  - `pinnedCount()` → `bool2int(netbirdAvail) + bool2int(warpAvail)`
  - `warpSelected()` → `warpAvail && cursor == (netbirdAvail ? 1 : 0)`
  - `selectedConfig()` → index into configs after subtracting `pinnedCount()`
  - `netbirdSelected()` stays `netbirdAvail && cursor == 0` (still correct).
- One shared `renderPinnedItem(name, connected, index, maxWidth)` replaces the
  NetBird-specific `renderNetbirdItem` and serves both rows (kills the
  row-render duplication). Status panels stay per-backend — WARP's fields
  (mode, account type) genuinely differ from NetBird's (peers, FQDN).

### Config list (left panel)

- When `exec.LookPath("warp-cli")` succeeds, a pinned **WARP** row appears
  below the NetBird row (or at the top if NetBird is absent), above WG configs.
- Not installed → row absent, nothing changes for existing users.
- Same `●` connected indicator as configs.
- Enter/space connects (`warp-cli connect`); `d` disconnects
  (`warp-cli disconnect`).
- Rename (`r`), delete (`x`) on the row flash "WARP is managed by its own
  daemon" and do nothing — same guard as NetBird.

### Coexistence (WARP vs WireGuard)

Unlike NetBird (a mesh overlay that only routes peer subnets and genuinely
coexists with a full tunnel), **WARP is itself a full-device tunnel** in its
default mode. Two full tunnels both claiming the default route + DNS conflict —
the OS lets both interfaces come up, but in practice one wins the route and the
other's traffic breaks. They only cleanly coexist when at least one is
split-tunnel (WARP split-tunnel, WARP DoH-only/proxy mode, or a WG config with
narrow `AllowedIPs`).

**Decision (open for review):** the tool treats WARP like NetBird —
**independent state, no enforced mutual exclusion.** `activeVPN` stays WG-only;
`warpStatus` tracks WARP separately, refreshed on the 1-second `statusTick` via
`warp-cli -j status`. Connecting WARP does **not** auto-disconnect an active WG
tunnel (and vice versa). Rationale: split-tunnel setups are legitimate, so a
hard block would be wrong; the tool reports daemon truth rather than enforcing
routing policy. The conflict is captured as a CLAUDE.md gotcha.

> Alternative if we want it: treat WARP as mutually exclusive with WG (connecting
> one disconnects the other, like switching between WG configs). More "correct"
> for full-tunnel users, wrong for split-tunnel users, and crosses the
> WG/daemon boundary. Not chosen unless the user prefers it.

### Status panel (right panel)

- Cursor on the WARP row → WARP details: status badge, plus whatever of
  {mode, account type, endpoint, transfer down/up} the JSON exposes,
  rendered defensively.
- Transfer/stats come from `warp-cli -j stats` (a separate subcommand, unlike
  NetBird which carries transfer in the status payload); rendered only if
  present.
- Cursor on a config row or the NetBird row → existing behavior, unchanged.

### Registration / daemon edge cases (the awareness-only fallbacks)

The `WarpStatus` parser must distinguish these semantic states (literal keys
pinned at impl):

- **Connected** — WARP tunnel up. Row shows `●`; status panel shows details.
- **Disconnected** — registered but not connected (the normal idle state).
  Connect is allowed.
- **Needs registration / SSO** — no valid registration. Do **not** run
  `warp-cli connect` (consumer registration is non-interactive, but Zero Trust
  Teams enrollment blocks on browser SSO; we mirror the NetBird guard and back
  off rather than guess). Flash / panel: "WARP needs registration — enroll in
  a terminal." Detection mechanism (`status` field vs a `registration` /
  `account` query) is pinned at impl.
- **Daemon down** — `warp-cli` errors with `code: FailedToConnectToDaemon`.
  Row still shows; status panel says the daemon isn't running with a
  `sudo systemctl enable --now warp-svc` hint (mirrors NetBird's daemon-down
  panel). Toggle errors surface as flash messages.

`connectWarp()` mirrors `connectNetbird()`: already-connected → "already
connected"; needs-registration or daemon-down → warn and return without
acting; otherwise enter `modalConnecting` and run `warp-cli connect`.

### Title bar & waybar

- Title bar shows every active connection. With WARP added the existing
  N-way switch becomes combinatorial, so it's refactored into a **dynamic
  badge builder**: append `● <name>` for each of {active WG, NetBird, WARP}
  that is up; if none, show `○ disconnected`. (Targeted refactor justified by
  the third state.)
- `--waybar` already builds a `parts` slice and appends per-source, so WARP is
  an additive block: class `connected` if **any** of WG / NetBird / WARP is up;
  tooltip lists each. No refactor needed there.

## Components

| File | Change |
|------|--------|
| `warp.go` (new) | `WarpAvailable()`, `WarpStatus` struct + `GetWarpStatus()` (parses `warp-cli -j status`, defensive), optional `GetWarpStats()`, `WarpUp()`, `WarpDown()`, state predicates `Connected()` / `NeedsRegistration()` / `DaemonDown()` |
| `model.go` | `warpAvail`/`warpStatus` fields; `pinnedCount()` + `warpSelected()` helpers + generalized `selectedConfig()`; `warpDoneMsg`; tick refresh; connect/disconnect + rename/delete routing; `connectWarp()` |
| `config_panel.go` | `renderPinnedItem()` (generalized from `renderNetbirdItem`); render WARP row; border-connected check includes `warpStatus.Connected()` |
| `status_panel.go` | `renderWarpStatusPanel()` + dispatch when `warpSelected()` |
| `dashboard.go` | Title bar dynamic badge builder (WG + NetBird + WARP) |
| `main.go` | `printWaybarStatus()` appends a WARP block |
| `warp_test.go` (new) | `warp-cli -j status` parse fixtures: connected, disconnected, needs-registration, daemon-down |
| `model_test.go` | Cursor/index helpers with two pinned rows (NetBird + WARP + configs) — the highest-risk surface |

## Testing

- **Unit:** `GetWarpStatus` parsing from JSON fixtures for each semantic state;
  `pinnedCount` / `warpSelected` / `selectedConfig` / `netbirdSelected` across
  the combinations {neither, NetBird only, WARP only, both} × cursor positions.
- **Manual:** TUI with `warp-cli` absent (row hidden, NetBird-only behavior
  unchanged); with WARP present and daemon down; connect/disconnect against a
  live registered WARP.

## Implementation-time verifications

1. Capture a real connected `warp-cli -j status` / `warp-cli -j stats` payload
   (start `warp-svc`, or pull from Cloudflare docs) and pin the JSON keys +
   the registration-needed detection.
2. Confirm `warp-cli connect` / `disconnect` run without sudo as the user. If
   they don't, add a sudoers rule via PKGBUILD as the WG path does.

## Out of scope

- WARP registration / Teams SSO enrollment from inside the TUI.
- Mode switching (`warp` / `doh` / `proxy`), trusted networks, WARP settings.
- Managing the `warp-svc` systemd unit.
- Enforced WG-vs-WARP conflict detection (documented gotcha only, unless the
  coexistence decision above is flipped to mutual-exclusion).
- A generic pinned-backend abstraction (approach A) — revisit only if a third
  daemon-managed VPN is added.
