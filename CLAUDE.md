# EmilyOS

EmilyOS is a Linux-based operating environment implementing the EmilyOS design philosophy: posture-gated sessions, capability-checked agency verbs, tamper-evident audit logging, and SOC 2-ready controls. It is not a bare-metal OS — it is the *policy kernel* that runs on Linux and enforces the invariants described in the legacy docs.

## North Star

SOC 2 Type II readiness. Every design decision is evaluated against this. See `docs/NORTHSTAR.md`.

## Stack

- Go 1.22+ — policy kernel, audit log, verb dispatcher, RBAC
- Linux (Ubuntu 22.04 LTS or Debian 12 baseline)
- systemd for domain lifecycle management
- No external databases — append-only JSONL audit log, JSON policy snapshots

## Key concepts

- **Posture**: the current operating mode of the system. NORMAL / SIEGE / MERCY / INCIDENT / GAME. Stored in `var/posture.json`.
- **Verb**: a declared intent. ENTER / PAUSE / RESUME / WITHDRAW / EXIT / GAME / SSH / INCIDENT. Every verb is capability-checked and audited.
- **Capability**: a named permission (e.g. `cap.net`, `cap.exec`, `cap.policy.write`). Granted to roles; roles assigned to identities.
- **Audit event**: an immutable record with hash chain. Every verb call emits one. See `internal/audit/`.
- **Policy snapshot**: a hash-addressed JSON file capturing the current RBAC config. Written on every policy change.

## Directory layout

```
cmd/emilyos/        -- entry point
internal/audit/     -- hash-chained append-only audit log
internal/policy/    -- RBAC roles + capability gates + policy snapshots
internal/posture/   -- posture state machine
internal/verb/      -- verb dispatcher
docs/               -- golden docs (NORTHSTAR, ARCHITECTURE, SOC2, etc.)
var/                -- runtime state (gitignored except .gitkeep)
```

## Build

```sh
go build ./cmd/emilyos
./emilyos --help
```

## Related repos

- `github.com/emilyspringerton/EMILY` — Emily Prime agent (RSI loop, cron, Apples)
- `github.com/emilyspringerton/IDUNA` — IAM + Apples store
- `github.com/emilyspringerton/MJOLNIR` — Android intelligence terminal

## Commit Protocol (standing instruction)

Always commit and push completed work immediately — don't wait to be asked. This is the default for every repo in this monorepo.
