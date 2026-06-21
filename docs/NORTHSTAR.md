# EmilyOS — North Star

**SOC 2 Type II Security (Common Criteria) Readiness**

> An operating environment that a Type II auditor can attest to: deliberate interaction, least privilege, isolation boundaries, tamper-evident audit trail, controlled change management.

---

## Why SOC 2 Is the Right North Star

SOC 2 doesn't tell us to build bureaucracy. It tells us to build the *right thing*:

| SOC 2 Requirement | EmilyOS Mechanism |
|---|---|
| Deliberate user action | Intent-declared verb model (no single-click accidents) |
| Least privilege | RBAC + capability gates on every verb |
| Logical access control | Operator / Admin / Auditor roles |
| Audit logging | Hash-chained append-only JSONL with all SOC 2 fields |
| Change management | Policy snapshots, versioned, hash-addressed |
| Incident response | INCIDENT posture, evidence export bundle |
| Monitoring | Audit trail queryable; alert hooks for tamper detection |

SOC 2 is the frame that makes every design decision answerable: *does this make the system more auditable, more deliberate, more controllable?*

---

## Milestones

### Milestone 1 — Audit Foundation
**Status:** `[x] complete` — 2026-06-15  
**Target:** Sprint 1 (2026-06-09 → 2026-06-16)

**Acceptance Criteria:**
- [x] `internal/audit/log.go`: append-only JSONL log with hash chain
- [x] Event schema matches SOC 2 §2.2: `ts, actor_id, session_id, device_id, verb, object_ref, decision, reason_code, result, prev_hash`
- [x] Tamper detection: `VerifyChain()` returns error on modified event
- [x] `audit_test.go`: 100 events written → tamper event 50 → chain invalid detected
- [x] Log is append-only: no update/delete methods exist in the interface

**Why this is first:** Nothing is trustworthy without an audit trail. Everything else is built on top of this.

---

### Milestone 2 — RBAC + Capability Gates
**Status:** `[x] complete` — 2026-06-15  
**Target:** Sprint 2

**Acceptance Criteria:**
- [x] `internal/policy/rbac.go`: Operator / Admin / Auditor roles with fixed capability sets
- [x] Capabilities: `cap.net`, `cap.exec`, `cap.policy.write`, `cap.audit.read`, `cap.export`, `cap.ssh.connect`, `cap.domain.start`, `cap.domain.stop`
- [x] `internal/verb/dispatch.go`: every verb checks capability before executing
- [x] Denied actions emit `decision=deny + reason_code` audit event
- [x] Auditor role: can read audit logs, cannot execute any verb that mutates state
- [x] Test: Auditor session denied on `cap.exec` attempt; audit event written

**Why this is second:** Capability gates without audit logs are silent. Audit logs without capability gates are incomplete.

---

### Milestone 3 — Posture State Machine
**Status:** `[x] complete` — 2026-06-15  
**Target:** Sprint 3

**Acceptance Criteria:**
- [x] `internal/posture/machine.go`: NORMAL / SIEGE / MERCY / INCIDENT / GAME / EXITED states
- [x] Posture transition rules enforced (see `docs/POSTURE.md`)
- [x] Each posture has a capability override mask
  - SIEGE: `cap.net = OFF` (hard gate regardless of role)
  - MERCY: `cap.exec` only allows pinned domains
  - INCIDENT: `cap.exec = DENY`, `cap.export = ALLOW`
  - GAME: `cap.net = OFF`, `cap.domain.start = DENY` except game domain
- [x] Posture transitions are audited (actor, old_posture, new_posture) — via log.Allow with from/to meta
- [x] Posture persists across restarts via `var/posture.json`
- [x] Test: SIEGE mode rejects `cap.net` capability even for Admin role (TestCapabilityVerdict_SiegeDeniesNet)

**Why this is third:** RBAC tells you who can do what. Posture tells you what the system allows right now.

---

### Milestone 4 — Policy Snapshots
**Status:** `[x] complete` — 2026-06-21  
**Target:** Sprint 4

**Acceptance Criteria:**
- [x] `internal/policy/snapshot.go`: hash-addressed JSON policy snapshot on every RBAC/capability change
- [x] Snapshot schema: `{snapshot_id, git_commit, build_id, created_at, actor_id, prev_snapshot_id, roles, capabilities, hash}`
- [x] `POLICY_ROLLBACK(snapshot_hash)` is an explicit verb, capability-checked (`cap.policy.write`), audited
  - `emilyos snapshot rollback <id>` dispatches verb via Dispatcher, verifies hash, emits audit event
- [x] Build attestation: `build_id` and `git_commit` embedded in binary at build time via `ldflags`
- [x] `emilyos about` shows build attestation in read-only pane
- [x] Test: 3 policy changes → rollback to snapshot 1 → verify RBAC matches snapshot 1 (TestSnapshotRollback)

---

### Milestone 5 — Evidence Export
**Status:** `[x] complete` — 2026-06-21  
**Target:** Sprint 5

**Acceptance Criteria:**
- [x] `EXPORT_EVIDENCE` verb (requires `cap.export`) — dispatched before bundle creation (S54-02)
- [x] `emilyos audit export <outdir>` produces `audit.jsonl` + `manifest.json` (S47-03)
- [x] Bundle is a `.tar.gz` with a `manifest.json` listing all files + their SHA-256 hashes (S50-02)
- [ ] `manifest.json` is signed if `EMILY_SIGNING_KEY` is present — deferred (no signing key in prod)
- [x] Export event is itself audited — EXPORT_EVIDENCE audit event emitted before bundle write
- [x] Test: `TestBundleManifestVerification` — builds synthetic bundle, extracts, verifies all SHA-256 entries

---

### Milestone 6 — SOC 2 Type II Readiness
**Status:** `[ ] future`  
**Target:** 90-day audit period after Milestone 5

**Acceptance Criteria:**
- [ ] All Milestones 1–5 complete
- [ ] 90-day continuous operation with consistent audit trail
- [ ] Session timeout (policy-defined idle lock) implemented and audited
- [ ] RBAC changes during the period are reflected in policy snapshot chain
- [ ] Evidence export bundle produced at audit start date
- [ ] Engage external SOC 2 auditor

---

## Design Constraints (Never Compromise)

These come directly from the legacy docs. They are not features. They are invariants.

1. **No single-click actions.** Every action requires declared intent. This is a SOC 2 control: "accidental actions are reduced."
2. **Audit before execution.** Permission check → audit event → execution. Never audit after the fact.
3. **Denied actions are audited.** A silent deny is an audit gap.
4. **Posture overrides role.** SIEGE can always override Admin-granted `cap.net`. Posture is physics; RBAC is policy.
5. **Log is append-only.** No update, no delete, no rotation without a new log and an audit event linking them.
6. **Absence is defended.** If something is not implemented, the README says why and what prerequisite it depends on.
7. **Stillness > throughput.** The daemon sleeps when idle. `time.Sleep` is not a failure mode.

---

## Version

`v0.1 — 2026-06-09 — Emily Prime (initial translation from legacy docs)`
