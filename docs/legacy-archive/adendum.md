EmilyOS 2 GUI Addendum v0.1 — SSH Pure Black Rule
 
emilyspringerton edited this page on Feb 3 · 1 revision
Color law: #000000 is SSH-only
Pure black (RGB 0,0,0 / #000000 / “kkkkkk”) may appear only inside an SSH Terminal pane.

Any other surface (background, panes, tiles, borders, overlays, HUD) must be near-black, not pure black.

If a renderer would naturally clear to black, it must clear to NEAR_BLACK instead.

Compliance test: sample pixels outside SSH panes — if any are #000000, it’s a failure.

SSH is a first-class verb and a first-class pane type 2.1 New verb: VERB_SSH
VERB_SSH toggles an SSH pane (same “intent declared” rules).

SSH pane is a domain under policy control, not a loose app.

2.2 SSH pane types

SSH Terminal Pane (pure black background)

SSH Host Selector Pane (NOT pure black — uses near-black like the rest of UI)

Interaction rules (consistent with no single-click)
Single click: focus only (never executes)

Fast double click (ACTIVATE):

on a host tile → open SSH terminal to that host (or bring existing one forward)

inside terminal → no “double click selects word then auto-copy”; keep it terminal-native but do not trigger OS actions

Slow double click (EDIT):

on host label → edit host alias / notes

on terminal title tab → rename pane label (not the host)

SSH configuration model (auditable)
Each SSH host entry is a tile/object:

host_id (hash-addressed)

alias (label)

user@host:port

auth_method:

key-based only by default

password allowed only if policy permits

key_ref (reference to key slot, not key material)

Policy: default deny adding new hosts unless Admin capability.

Key handling (SOC 2 aligned)
Private keys are never displayed.

Keys are stored in a protected store (implementation TBD), referenced by key_ref.

Any of:

key add/remove

host add/remove

first connection trust decision (known_hosts) …must be audited.

Audit events (minimal):

ssh.host.add/remove/edit

ssh.key.add/remove/rotate

ssh.connect.start/stop

ssh.trust.accept/deny (host key verification result)

Never log secrets, never log private key material.

SSH terminal rendering rules
Inside the SSH terminal pane:

background = pure black

text colors allowed:

ANSI palette (but still “no pure white” rule can be relaxed inside terminal if you want maximum legibility)

selection, cursor, caret:

permitted, but must not animate beyond standard terminal cursor blink (and you can disable blink for stillness)

Outside terminal: no ANSI rainbow dumping onto the main UI.

Layout integration (tmux × i3)
SSH panes behave like any other pane:

splittable, resizable, focusable

cannot float

can be pinned (optional) so posture changes won’t auto-close it

But posture rules still apply:

MERCY: SSH allowed to exist but default deny new outbound connections (no reaching outward) unless explicitly permitted.

SIEGE: SSH connections denied by default (network hard-off) unless policy grants cap.net.ssh.

Capabilities (explicit)
Add:

cap.ssh.open_pane

cap.ssh.connect

cap.net.ssh

cap.ssh.manage_hosts

cap.ssh.manage_keys

Defaults:

Operator: open pane + connect (if net allowed)

Admin: manage hosts/keys

Auditor: read-only host list + connection logs (no connect)

Visual identity rule
SSH is visually distinct by the black:

if you see pure black, you are “in terminal reality.”

it becomes a psychological boundary: “this is raw system contact.”
