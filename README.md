# EmilyOS — Policy Kernel

EmilyOS is **not** a bare-metal OS. It is a Go *policy kernel* that runs on Linux and enforces
posture-gated sessions, capability-checked agency verbs, and tamper-evident (hash-chained) audit
logging — the SOC 2 Type II-readiness invariants described in `docs/NORTHSTAR.md`. The earlier
bare-metal exokernel framing (and the tile-based "GUI v0.1" filesystem-browser design that used to
live in this README) is superseded; see `docs/legacy-archive/` for that history.

## Current direction (2026-08-25/26)

Founder real-time: *"oh yea cool lets build our own installable distro"* → *"EmilyOS for real as
arch linux"* → *"PARENA native as much as possible"* → *"we use gnu tools for the load bearing
walls like grep"* → *"but the stuff we dont need leave it out"* → *"make it super small to start
whatever can be left out leave it out"* → *"build parena in"* → *"and vim"* → *"and emily cli."*
This is a scoping-only pivot toward a real, small, installable Arch-based Linux distro with
EmilyOS's policy kernel as the base, PARENA as the native language wherever practical, GNU
coreutils kept only where they're load-bearing (grep, etc.), and PARENA/vim/emily-cli built in.
Not yet built — see `docs/NORTHSTAR_DISTRO.md` (Draft v0.1) for the full scoping doc. This does
not change anything below; the policy kernel itself is unaffected by the pivot.

## Build & run

```sh
make build          # dist/emilyos, ldflags-stamped with git commit + build date
make build-static   # static linux/amd64 binary
make test           # GOWORK=off go test ./...
make verify          # build + test
make deb             # .deb package
```

```sh
./dist/emilyos --help
```

## CLI surface

```
emilyos posture get                    print current posture state
emilyos posture set <state>            transition posture (admin only)
emilyos verb dispatch <verb> <object>  dispatch a capability-checked verb
emilyos audit tail [-n N]              print last N audit events (default 10)
emilyos audit verify                   verify audit hash-chain integrity
emilyos audit export / bundle / history
emilyos snapshot capture|list|show <id>|rollback <id>   RBAC policy snapshots
emilyos fs grant <identity> <path>     grant filesystem ACL access (see below)
emilyos fs revoke <identity> <path>    revoke filesystem ACL access
emilyos about                          version / build attestation
```

Environment: `EMILY_ACTOR_ID`, `EMILY_SESSION_ID`, `EMILY_DEVICE_ID`, `EMILY_ROLE`
(operator|admin|auditor), `EMILY_POSTURE_PATH`, `EMILY_AUDIT_PATH`.

## Key concepts

- **Posture**: the system's current operating mode — NORMAL / SIEGE / MERCY / INCIDENT / GAME.
  Stored in `var/posture.json`. A Ravenscar Ada port of this state machine lives in `ada/posture/`.
- **Verb**: a declared intent (ENTER / PAUSE / RESUME / WITHDRAW / EXIT / GAME / EXEC /
  DOMAIN_EXEC / NET / SSH / INCIDENT / EXPORT / POLICY_CHANGE / AUDIT_READ / DOMAIN_START /
  DOMAIN_STOP / SSH_MANAGE_HOSTS / SSH_MANAGE_KEYS / GRANT_FS / REVOKE_FS). Every verb is
  capability-checked and audited.
- **Capability**: a named permission (e.g. `cap.net`, `cap.exec`, `cap.policy.write`). Granted to
  roles; roles assigned to identities.
- **Audit event**: an immutable, hash-chained record. Every verb call emits one — `internal/audit/`.
- **Policy snapshot**: a hash-addressed JSON file capturing current RBAC config, written on every
  policy change — `internal/policy/`.

## GRANT_FS / REVOKE_FS — real filesystem ACLs (2026-08-25)

`emilyos fs grant <identity> <path>` / `emilyos fs revoke <identity> <path>` dispatch the
`GRANT_FS`/`REVOKE_FS` verbs, which are capability-checked, posture-gated, and audited like any
other verb — replacing hand-run `setfacl` invocations (e.g. `sudo-queue/22-create-treeiii-user.sh`)
with declared policy. The actual ACL mutation goes through `internal/fsaclmod`, a real
PARENA-compiled mod — the same "mod is the trigger, host Go/C does the real work" shape PITVIPER's
`scrollmod` established (S192-01). This is EmilyOS's first real PARENA integration point.

## Directory layout

```
cmd/emilyos/        -- entry point, CLI dispatch
internal/audit/     -- hash-chained append-only audit log
internal/policy/    -- RBAC roles + capability gates + policy snapshots
internal/posture/   -- posture state machine
internal/verb/      -- verb dispatcher
internal/fsacl/     -- filesystem ACL policy types
internal/fsaclmod/  -- PARENA-compiled mod driving the real setfacl-equivalent calls
ada/posture/        -- Ravenscar Ada port of the posture state machine
docs/               -- golden docs (NORTHSTAR, NORTHSTAR_DISTRO, ARCHITECTURE, SOC2, FS_ACL_COORDINATION, etc.)
docs/legacy-archive/-- superseded design docs, kept for historical reference only
var/                -- runtime state (gitignored except .gitkeep)
```

## Documentation

- `docs/NORTHSTAR.md` — SOC 2 Type II readiness north star (written by Emily Prime, 2026-06-09)
- `docs/NORTHSTAR_DISTRO.md` — Arch-Linux-distro pivot scoping doc (Draft v0.1, 2026-08-25)
- `docs/ARCHITECTURE.md` — policy-kernel-on-Linux architecture (vs. the superseded bare-metal design)
- `docs/FS_ACL_COORDINATION.md` — GRANT_FS/REVOKE_FS design rationale
- `docs/legacy-archive/` — superseded exokernel/GUI design docs; do not use for planning or implementation

## Related repos

- `EMILY` — Emily Prime agent (RSI loop, cron, Apples)
- `IDUNA` — IAM + Apples store
- `MJOLNIR` — Android intelligence terminal
- `PITVIPER` — established the PARENA-mod integration shape (`scrollmod`) that `fsaclmod` follows
- `PARENA` — the language `fsaclmod` is compiled from
