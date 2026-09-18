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

## Founder Real-Time Direction

Whenever the founder gives real-time direction — a new ask, a correction, a "can we also..." —
route it through `emily observe -s info "Founder real-time: <summary>"` first, even if it isn't
this repo's usual domain, then sprint-plan it into `EMILY/BACKLOG.md` (`emily backlog curate`,
scoped into a real SECTION/sub-item, not just a one-line log), and only then implement. See
`EMILY/docs/THE_EMILY_WAY.md` Principle 18 ("Pave the Cow Paths").

## README Reality — SAGA reconciliation (standing instruction, monorepo-wide)

Founder real-time, 2026-09-18: if a change of yours **substantially changes the claim of this project's core README**,
then per SAGA protocols (`EMILY/docs/SAGA_SYSTEM_AUDIT_2026-07-18.md`, HQ-SPEC-DOC-102: intent ↔ claim ledger ↔ reality)
you **must update `README.md` in the same unit of work** so it reflects current reality. The README is the project's public
claim; it must not lag behind the code.

- **When it applies:** a capability is added or removed; status moves ("design only" → "working", "planned" → "shipped");
  the stack, build, run or install steps change; a claim in the README is now false or stale; or you add a **meaningful,
  genuinely interesting piece of kit** (a new tool, engine capability, protocol, pipeline, game system). For that last case
  especially: put it in the README — what it is, how to run it, and its honest status and limits.
- **When it does not:** ordinary fixes, refactors and small features that leave the README's claims true.
- **How:** re-read the README against what you just changed; fix or delete stale lines (including "not built yet" notes that
  are now built); verify any new claim by actually running it, and mark anything untested as untested; commit the README
  with (or immediately after) the change, and mention it in the CHANGELOG entry.

## Frame-Break Reframing

Founder-sourced prompting technique (REDGARDEN/NORTHSTAR.md §28, full origin in
REDGARDEN/docs2/MULTI_AGENT_RD_RESEARCH_NOTES.md §5): given a request, name the underlying
structural/systemic pattern it's one instance of — one level of abstraction up — as an added
lens during planning/triage/judgment calls. Use it to spot the general case behind a specific
ask. It augments judgment, it does not replace doing the work: direct, concrete execution of
the literal task asked for still happens every time.

## Commit Protocol (standing instruction)

Always commit and push completed work immediately — don't wait to be asked. This is the default for every repo in this monorepo.

Every commit — human-written or produced by automated code paths (git-commit helpers in emily-agent, emily.cli, IDUNA handlers, etc.) — must carry the active `emily session` fingerprint as a `session: <tag>` trailer (blank line, then the trailer). This was silently missing from several independently-implemented automated commit helpers across the monorepo until an audit on 2026-08-10 (founder, real-time: "where in the fuck is my llm session id anywhere"). If you add a new automated git-commit code path anywhere, wire in the session tag the same way — don't assume an existing helper already does it.
