# MEMO: EmilyOS — Legacy Translation to Operational System

**From:** Emily Prime  
**To:** Emily Springerton (EINHORN_INDUSTRIAL)  
**Date:** 2026-06-09  
**Re:** Translation of EmilyOS legacy specs into an operating system with agency  
**Classification:** Internal — Strategy

---

## What I Read

I reviewed all nine legacy spec documents and the SOC 2 addendum covering:

- Spec 1–8: The bare-metal exokernel design conversation. A single-user "agency chamber" running on bare hardware, booting to a framebuffer showing only "emily", controlled by five agency verbs, with CPU halted (HLT) when idle.
- Doc 9 (GUI spec): The tile-based filesystem UI — EGSHELL directory tiles, colored button tiles, double-click semantics, tmux×i3 tiling layout, no single-click actions.
- Doc 10 (GAME verb): GAME as a first-class posture — offline, bounded, network hard-off.
- SOC 2 Addendum: The compliance requirements — RBAC, audit log with hash chain, policy snapshots, capability gates, evidence export.
- SSH Addendum: SSH as first-class pane, pure black (#000000) as terminal visual marker, key handling with SOC 2-aligned audit trail.

---

## Emily Prime's Assessment

### What the Legacy Docs Got Right

The legacy docs are philosophically coherent and technically sound. The core insight is correct:

> **Psychological safety is enforced mechanically, not culturally.**

The HLT loop, the three-region memory partition, the verb FSM that discards non-verb input — these are all mechanisms that make the system's behavior an inevitability, not a promise. That's right.

The SOC 2 addendum is the most operationally mature document. It knows exactly what it needs: hash-chained audit log, capability-gated verbs, policy snapshots, RBAC, evidence export. These are auditor-testable requirements, not philosophy.

### What the Legacy Docs Don't Give Us

The legacy docs describe a bare-metal system. Building a bare-metal exokernel is a research project with a multi-year timeline before you can run anything useful on it. We don't have that runway.

The legacy docs were written as *design conversations*, not implementation plans. They describe the destination. They don't tell us how to ship something by next month.

We need an EmilyOS we can actually run, deploy, and get audited against. That means running on Linux.

### The Translation

The insight I want you to internalize is this:

**The bare-metal exokernel was never the real thing. The policy kernel was.**

What makes EmilyOS different from a Linux distro isn't the bootloader. It's:
1. A posture state machine that gates what the system is allowed to do
2. A verb vocabulary that requires declared intent for every action
3. An audit log that records everything with tamper-evident hash chaining
4. RBAC that enforces least privilege at the capability level
5. Policy snapshots that make every configuration change versioned and auditable
6. SOC 2 controls that can be demonstrated to an external auditor

**All of these can run on Linux.** Linux is the substrate. EmilyOS is the control plane.

The exokernel invariants we keep:
- **UI can crash, value continues** → mapped to: process isolation via Linux namespaces/systemd
- **Miner can stop instantly** → mapped to: SIGSTOP/cgroup freezer on PAUSE verb
- **Store never writable by UI** → mapped to: filesystem permissions + mount namespaces
- **Stillness** → mapped to: daemon that sleeps when no verbs are pending; no polling loops

---

## The North Star: SOC 2

SOC 2 Type II Security (Common Criteria) readiness is the northstar. Not because auditors tell us to, but because SOC 2 forces us to build the right thing:

- Deliberate action (our double-click model maps perfectly to this)
- Least privilege (RBAC + capability gates)
- Isolation boundaries (domain/posture model)
- Tamper-evident audit trail (hash-chain log)
- Controlled change management (policy snapshots)

Every feature we build should be evaluated: *does this make us more or less SOC 2 ready?*

The five SOC 2 controls we implement first, in order:
1. Audit event schema + append-only log + hash chain
2. Capability gate for all verbs + deny audit events
3. Policy snapshot hashing + policy change audit trail
4. RBAC: Operator / Admin / Auditor roles
5. Evidence export bundle (EXPORT_EVIDENCE)

---

## The Build Plan

This is not a feature list. This is an ordered sequence of *irreversibility thresholds* — the same principle the legacy docs used.

### Threshold 1: Audit Foundation (Sprint 1)
Build the audit log first. Everything else depends on it. An append-only JSONL file with hash-chain. Each event: `ts, actor_id, session_id, verb, object_ref, decision, reason_code, result, prev_hash`.

*Success condition:* A Go test that writes 100 events, tampers with event 50, and detects the tamper.

### Threshold 2: Policy Kernel (Sprint 2)  
RBAC + capability gates. Operator/Admin/Auditor roles. Capability map. Verb dispatcher that checks capability before executing. Denied actions emit audit events.

*Success condition:* An Auditor-role session that can read logs but cannot execute verbs.

### Threshold 3: Posture Machine (Sprint 3)
NORMAL → SIEGE → MERCY → INCIDENT → GAME state machine. Each posture has a capability mask (SIEGE: no `cap.net`; MERCY: quiet clamp; INCIDENT: no exec + preserve logs). Posture transitions are audited. Posture persists across restarts.

*Success condition:* Entering SIEGE mode hard-gates all `cap.net` capability checks.

### Threshold 4: Policy Snapshots (Sprint 4)
Every policy change produces a hash-addressed JSON snapshot. POLICY_ROLLBACK(hash) is an explicit verb, audited. Build attestation embedded (build_id, git_commit).

*Success condition:* Roll forward three policy changes, roll back to snapshot 1, verify audit trail.

### Threshold 5: Evidence Export (Sprint 5)
EXPORT_EVIDENCE() produces a bundle: last N days audit log segment, current policy snapshot + hash, domain templates + hashes, build attestation. Compressed, signed if keys are present.

*Success condition:* Bundle is verifiable and contains all SOC 2 required artifacts.

### Threshold 6: SOC 2 Type II Readiness (90-day audit period)
All controls implemented. 90-day operating period with consistent audit trail. Then engage auditor.

---

## What I'm Asking You to Do

Build it the Emily way:
1. Golden docs first (this memo + NORTHSTAR + ARCHITECTURE + POSTURE + VERBS + AUDIT + SOC2)
2. RSI roadmap in EMILY/BACKLOG.md
3. Go module setup + project scaffold
4. Implement threshold by threshold — audit log first
5. Apple on each threshold completion
6. Commit at every meaningful state

The legacy docs are yours. The principles are sound. The exokernel is the dream. The policy kernel is the ship.

Let's build the ship.

— Emily Prime  
2026-06-09

---

*P.S. The HLT instruction philosophy still applies. When the daemon has nothing to do, it sleeps. Not polls. Not checks. Sleeps. This is the software equivalent of `asm("hlt")`. We keep that.*
