# EmilyOS — Architecture

**Version:** 0.1  
**Author:** Emily Prime  
**Date:** 2026-06-09  

---

## The Translation Principle

The legacy docs designed a bare-metal exokernel. We build a *policy kernel* on Linux.

The exokernel's job was to enforce three resource partitions physically:

| Domain | Resource | Kernel Guarantee |
|---|---|---|
| UI | Framebuffer + input | Can die without touching others |
| Miner | CPU cycles + timers | Can be halted instantly |
| Store | Non-volatile memory | Never writable by UI |

We map these invariants to Linux mechanisms:

| EmilyOS Invariant | Linux Mechanism |
|---|---|
| UI crash → value continues | systemd services with isolation; process namespaces |
| Miner instant pause | SIGSTOP or cgroup v2 freezer on PAUSE verb |
| Store not writable by UI | Mount namespaces; file ACLs; separate UID |
| Stillness (HLT) | `select{}` / `time.Sleep`; no polling loops |

---

## System Layers

```
┌─────────────────────────────────────────────────────────┐
│  IDENTITY LAYER                                         │
│  OperatorID / ServiceID — every session has one         │
│  No anonymous actions. UNTRUSTED_MODE banner if absent. │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│  VERB DISPATCHER                                        │
│  All user/agent actions enter as declared verbs.        │
│  Verbs: ENTER PAUSE RESUME WITHDRAW EXIT GAME SSH       │
│         INCIDENT POLICY_CHANGE POLICY_ROLLBACK          │
│         DOMAIN_START DOMAIN_STOP DOMAIN_EXEC EXPORT     │
│  Each verb: capability check → audit event → execute    │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│  POSTURE MACHINE                                        │
│  Current operating mode: NORMAL / SIEGE / MERCY /       │
│                          INCIDENT / GAME                │
│  Posture overrides RBAC for certain capabilities.       │
│  Posture persists across restarts.                      │
└───────────┬─────────────┬───────────────────────────────┘
            │             │
┌───────────▼──────┐  ┌───▼────────────────────────────────┐
│  POLICY KERNEL   │  │  AUDIT LOG                         │
│  RBAC roles      │  │  Append-only JSONL                 │
│  Capability map  │  │  Hash-chained (prev_hash)          │
│  Policy snapshot │  │  Tamper-evident                    │
│  versioning      │  │  Never writable by UI              │
└───────────┬──────┘  └────────────────────────────────────┘
            │
┌───────────▼──────────────────────────────────────────────┐
│  DOMAIN LAYER (Linux substrate)                          │
│  systemd services + namespaces + cgroups                 │
│  Domains: work / game / ssh / store                      │
│  Each domain has a capability template                   │
└──────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. Verb Dispatcher (`internal/verb/`)

Every action in EmilyOS is a verb. There are no implicit side effects.

```
Verb → Check capability → Emit audit event → Execute action
                ↓
          DENY if:
          - caller lacks capability
          - posture masks the capability
          → Emit deny audit event
          → Return ErrDenied
```

Verb table:

| Verb | Capability Required | SOC 2 Event |
|---|---|---|
| ENTER | `cap.session.open` | session.start |
| EXIT | `cap.session.open` | session.end |
| PAUSE | `cap.posture.set` | posture.change |
| RESUME | `cap.posture.set` | posture.change |
| GAME | `cap.posture.set` | posture.change |
| SSH | `cap.ssh.connect` + `cap.net` | ssh.connect |
| INCIDENT | `cap.posture.set` (Admin) | posture.change |
| DOMAIN_START | `cap.domain.start` | domain.lifecycle |
| DOMAIN_STOP | `cap.domain.stop` | domain.lifecycle |
| DOMAIN_EXEC | `cap.exec` | domain.exec |
| POLICY_CHANGE | `cap.policy.write` | policy.change |
| POLICY_ROLLBACK | `cap.policy.write` | policy.change |
| EXPORT | `cap.export` | export |
| AUDIT_READ | `cap.audit.read` | audit.access |

### 2. RBAC Roles (`internal/policy/`)

Three roles, fixed. Not extensible in v0 (simplicity = auditability).

| Role | Capabilities |
|---|---|
| Operator | `cap.session.open`, `cap.posture.set`, `cap.exec`, `cap.domain.start`, `cap.domain.stop`, `cap.ssh.connect`, `cap.net` |
| Admin | All Operator caps + `cap.policy.write`, `cap.export`, `cap.audit.read`, `cap.ssh.manage_hosts`, `cap.ssh.manage_keys` |
| Auditor | `cap.audit.read`, `cap.export` only |

### 3. Posture Machine (`internal/posture/`)

States and capability overrides:

| Posture | `cap.net` | `cap.exec` | `cap.domain.start` | Notes |
|---|---|---|---|---|
| NORMAL | role | role | role | All role capabilities apply |
| SIEGE | **OFF** | role | only-pinned | Network hard-off. Rationale: hostile outside. |
| MERCY | role | only-pinned | only-pinned | Quiet clamp; demands reduced |
| INCIDENT | **OFF** | **OFF** | **OFF** | Preserve + export only |
| GAME | **OFF** | game-domain-only | game-domain-only | Offline bounded space |

SIEGE's `cap.net = OFF` is a posture override. An Admin with `cap.net` is still denied in SIEGE. Posture is physics; RBAC is policy.

### 4. Audit Log (`internal/audit/`)

Each event is a JSONL record:

```json
{
  "ts": "2026-06-09T12:34:56.789Z",
  "seq": 42,
  "actor_id": "emily-springerton",
  "session_id": "sess-abc123",
  "device_id": "emily-prime-dev-01",
  "verb": "DOMAIN_EXEC",
  "object_ref": "domain:work/bin/claude-code",
  "decision": "allow",
  "reason_code": "",
  "result": "success",
  "prev_hash": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "hash": "sha256:a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
}
```

The `hash` field is SHA-256 of `(prev_hash + JSON of all other fields)`. `VerifyChain()` walks all events and detects any modification.

Log is stored at `var/audit.jsonl`. Rotated via explicit `AUDIT_ROTATE` verb (Admin only), with a linking event that records the new log path.

### 5. Policy Snapshots (`internal/policy/`)

On every `POLICY_CHANGE`:

```json
{
  "snapshot_id": "snap-2026-06-09T12:34:56Z",
  "created_at": "2026-06-09T12:34:56Z",
  "actor_id": "emily-springerton",
  "build_id": "emilyos-v0.1-20260609",
  "git_commit": "fe7cfe4",
  "prev_snapshot_id": "snap-2026-06-09T10:00:00Z",
  "roles": { ... },
  "capabilities": { ... },
  "posture_policy": { ... },
  "hash": "sha256:..."
}
```

Stored in `var/policy/snap-<timestamp>.json`. `POLICY_ROLLBACK(hash)` finds the snapshot and restores it, emitting a `POLICY_CHANGE` audit event with `rollback=true`.

---

## Domain Model

A domain is a systemd service (or LXC container in the full implementation) with a locked capability template:

| Domain | Template | Network | Filesystem |
|---|---|---|---|
| `work` | `work-default` | role-gated | overlayfs on shared home |
| `game` | `game-locked` | OFF | tmpfs only |
| `store` | `store-rw` | OFF | dedicated partition, no-exec |
| `ssh` | `ssh-pane` | via `cap.net.ssh` only | read-only |

Domain lifecycle events (start/stop/exec/snapshot) are all audited.

---

## What Is Intentionally Missing (v0)

| Missing | Why |
|---|---|
| Bare-metal boot | Linux is the substrate. Bare metal is a future R&D track. |
| LXC domain isolation | systemd services first; LXC comes after policy kernel is proven |
| GPU / framebuffer UI | TUI first; framebuffer UI is Milestone 7+ |
| SSH key management | Stub in v0; full key store in Milestone 4+ |
| Multi-user | Single operator per session; multi-user is a SOC 2 Availability extension |

Absence is defended. Nothing gets added until its prerequisite is proven.
