# EmilyOS — Filesystem ACL Coordination (design pass, not yet implemented)

## Why this exists

2026-08-25: a new scoped Linux account (`treeiii`) was created alongside `fatbaby` on the
EINHORN_INDUSTRIAL box, with real write access to `fatbaby`'s working tree via POSIX ACLs
(`sudo-queue/22-create-treeiii-user.sh`) — see `EMILY/BACKLOG.md` S191-06. The founder's own
concern, verbatim: *"but when fatbaby comes back the permissions cant be all fucked up"* — a
real, well-known multi-user Unix problem: a file either identity creates or resaves (many
editors save-via-rename, allocating a fresh inode) must remain writable by the other, or the
first identity to touch a shared file silently locks the second one out.

That script's fix — a recursive `setfacl` grant plus a default ACL on every directory covering
both identities — is correct but ad hoc: hand-run once, not tracked as policy, not audited, and
will not automatically extend to the next scoped account someone creates. Founder, immediately
after: *"we may need to develop our own software to handle this"* → *"put that work into
EmilyOs repo."* EmilyOS is already the policy kernel (RBAC, capability gates, hash-chained
audit log, policy snapshots per commit) — this is that same model applied to a new domain:
**which Linux identities can read/write which parts of the filesystem, as declared policy
instead of a script someone has to remember to re-run.**

This document is a design pass only. No code in this repo has been written yet.

## Fit with the existing model

`internal/policy/rbac.go` already has the extension point this needs, explicitly reserved:

> Additional capabilities can be granted per-identity via the identity override map (not yet
> implemented in v0 — reserved for Milestone 4).

Filesystem ACL coordination is a real, concrete instance of that reserved milestone, not a new
architectural concept. Proposed shape, following the existing `cap.*` naming convention
(`internal/policy/rbac.go`'s `CapSessionOpen`/`CapPostureSet`/etc.):

- `cap.fs.grant` — declare that identity X gets `mode` access to `path`, recursively, with a
  default-ACL entry so future files under `path` inherit it too.
- `cap.fs.revoke` — the inverse.
- `cap.fs.audit` — read the current effective grant set for a path or identity (a `getfacl`
  wrapper, not a new store — the filesystem ACLs *are* the source of truth; EmilyOS's own
  policy snapshot records *why* a grant was made and *who* authorized it, not a duplicate
  permission model that could drift from what the kernel actually enforces).

Only `RoleAdmin` gets `cap.fs.grant`/`cap.fs.revoke` in v0 (matches `CapPolicyWrite`'s existing
Admin-only scoping) — granting filesystem access is exactly as sensitive as writing policy.

## Proposed verb

A new verb, dispatched through the existing `internal/verb/dispatch.go` capability-checked path
(`ENTER`/`PAUSE`/`RESUME`/etc. are the existing precedent) — tentatively `GRANT-FS` /
`REVOKE-FS`. Each call:

1. Capability-checks the caller's role against `cap.fs.grant` (existing `HasCapability`).
2. Shells out to `setfacl`/`getfacl` (this is a thin, audited wrapper around the real kernel ACL
   mechanism — not a reimplementation of POSIX ACLs in Go).
3. Emits one hash-chained audit event per grant/revoke (`internal/audit/`, existing pattern) —
   *who* authorized *which identity* to access *which path*, *when*, closing the real gap the
   `sudo-queue/22` script has today: it's a one-time human action with no durable record beyond
   this NORTHSTAR doc and a BACKLOG entry.
4. Writes a policy snapshot (`internal/policy/snapshot.go`'s existing `Write`) capturing the
   current declared grant set, so `emilyos policy show`-style tooling can answer "what should be
   true" without re-deriving it from a live `getfacl` walk every time.

## What this deliberately does NOT do (v0 scope)

- **Not a permission-drift daemon.** v0 does not watch the filesystem and auto-reapply policy if
  someone runs a manual `chmod`/`setfacl` outside EmilyOS. That's real, larger scope (a `cap.fs`
  reconciliation loop) — flagged as a possible Milestone 5+ item, not attempted here.
- **Not a replacement for `sudo-queue`.** Anything requiring root (creating the Linux account
  itself, `usermod -aG`) still goes through the existing "write a script, human runs it with
  sudo" pattern — this only owns the ACL-grant half, which is the part that recurs and needs to
  stay in sync as new accounts/paths appear.
- **Not applied retroactively to `sudo-queue/22`** — that script stays as the real, working
  fix for `treeiii`/`fatbaby` specifically. This design is for the *next* account, so it doesn't
  need another hand-written script and a fresh round of "will this break fatbaby's own access"
  reasoning each time.

## Open questions (real, not decided here)

- Does `cap.fs.grant` take a raw filesystem path, or a symbolic "domain" (e.g. `fatbaby-tree`)
  that maps to a path — the latter matches EmilyOS's existing `domain.start`/`domain.stop`
  verbs' own vocabulary better, but needs a real domain registry that doesn't exist yet.
  Given fatbaby's already-established multiplication of scoped accounts (`secondtree`,
  `treeiii`), a symbolic name has a growing enumeration to draw on live rather than guessed.
- Exclusion policy (S191-06's own "the two secrets files and the SSH key stay owner-only" call)
  — should this be a per-grant explicit exclusion list, or a standing EmilyOS-wide denylist of
  path patterns (`*/var/*-secrets.env`, `~/.ssh/id_*`) that `cap.fs.grant` refuses to touch
  regardless of who asks? The latter is safer (a mistake in one grant call can't leak the crown
  jewels) and matches EmilyOS's "Design Constraints (Never Compromise)" section's existing
  posture — worth reading that section before implementation starts.
- Where does the *first* grant for a brand-new account's home directory (its own files, not
  the shared `fatbaby` tree) fit — is that in scope for `cap.fs.grant` too, or purely
  `useradd -m`'s job (which already handles it correctly on its own)?

## Status

Design only. Real implementation, tests, and the actual Go code in `internal/policy`/
`internal/verb` are unstarted — tracked as `EMILY/BACKLOG.md` S191-07.
