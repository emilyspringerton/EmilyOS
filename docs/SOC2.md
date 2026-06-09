# EmilyOS — SOC 2 Controls Map

**Version:** 0.1  
**Scope:** Security Trust Service Criteria (Common Criteria)  
**Author:** Emily Prime  

---

## Compliance Posture

Default SOC 2 scope: **Security (Common Criteria) only** — this is what we're building to.

Optional modules (enable per contract): Availability, Confidentiality, Privacy, Processing Integrity.

SOC 2 is an attestation about controls. We do not claim "unhackable." We claim:
- Deliberate interaction
- Least privilege
- Isolation boundaries  
- Tamper-evident audit trail
- Controlled change management

---

## Controls Map

### CC1 — Control Environment

| Control | EmilyOS Implementation | Status |
|---|---|---|
| CC1.1 — COSO Principle 1 (Integrity & Ethics) | Operator identity mandatory; UNTRUSTED_MODE banner if absent | `[ ] Milestone 2` |
| CC1.2 — Board/Management oversight | Emily Prime as meta-orchestrator; Apples are the oversight record | `[ ] Milestone 2` |
| CC1.3 — Organizational structure | Operator / Admin / Auditor role hierarchy | `[ ] Milestone 2` |
| CC1.4 — Competence commitment | Admin required for policy changes | `[ ] Milestone 2` |
| CC1.5 — Accountability | Every action tied to actor_id; no anonymous actions | `[ ] Milestone 1` |

### CC2 — Communication and Information

| Control | EmilyOS Implementation | Status |
|---|---|---|
| CC2.1 — Information quality | Audit log with hash chain = authoritative information record | `[ ] Milestone 1` |
| CC2.2 — Internal communication | MJOLNIR push notifications for posture changes | `[ ] Post M3` |
| CC2.3 — External communication | SOC 2 evidence export bundle for auditor | `[ ] Milestone 5` |

### CC3 — Risk Assessment

| Control | EmilyOS Implementation | Status |
|---|---|---|
| CC3.1 — Risk identification | Posture system (SIEGE/INCIDENT) = formal risk posture declaration | `[ ] Milestone 3` |
| CC3.2 — Risk analysis | INCIDENT posture clamps exec; preserves evidence | `[ ] Milestone 3` |
| CC3.3 — Fraud risk | TAMPER_ATTEMPT audit events; chain verification | `[ ] Milestone 1` |

### CC5 — Control Activities

| Control | EmilyOS Implementation | Status |
|---|---|---|
| CC5.1 — Controls selected | Capability gates on all verbs | `[ ] Milestone 2` |
| CC5.2 — Technology controls | Hash-chained audit log; policy snapshot versioning | `[ ] Milestone 1+4` |
| CC5.3 — Policies enforced | RBAC roles; posture overrides; deny audit events | `[ ] Milestone 2+3` |

### CC6 — Logical and Physical Access

| Control | EmilyOS Implementation | Status |
|---|---|---|
| CC6.1 — Access controls | RBAC: Operator/Admin/Auditor; session open/close audited | `[ ] Milestone 2` |
| CC6.2 — Authentication | Session must have OperatorID; AUTH_FAILURE audited | `[ ] Milestone 2` |
| CC6.3 — Role access management | PRIVILEGE_CHANGE events; Admin required for role changes | `[ ] Milestone 2` |
| CC6.6 — Logical access provisions | Capability gates; posture overrides; deny events | `[ ] Milestone 2+3` |
| CC6.7 — System access restriction | Domain isolation; namespace separation | `[ ] Post M3` |
| CC6.8 — Malicious software | Domain read-only rootfs; exec allowlist; INCIDENT posture | `[ ] Post M3` |

### CC7 — System Operations

| Control | EmilyOS Implementation | Status |
|---|---|---|
| CC7.1 — Infrastructure monitoring | Audit log review; evidence export | `[ ] Milestone 5` |
| CC7.2 — System operations | DOMAIN_* events; domain lifecycle audited | `[ ] Milestone 2` |
| CC7.3 — Anomaly detection | Chain verification; INTEGRITY_FAILURE events | `[ ] Milestone 1` |
| CC7.4 — Security incidents | INCIDENT posture; EVIDENCE_EXPORT verb | `[ ] Milestone 3+5` |
| CC7.5 — Incident recovery | POLICY_ROLLBACK; posture transition out of INCIDENT | `[ ] Milestone 4` |

### CC8 — Change Management

| Control | EmilyOS Implementation | Status |
|---|---|---|
| CC8.1 — Change mgmt process | POLICY_CHANGE + policy snapshot versioning; rollback audited | `[ ] Milestone 4` |

### CC9 — Risk Mitigation

| Control | EmilyOS Implementation | Status |
|---|---|---|
| CC9.1 — Risk mitigation | Posture machine; domain isolation; capability gates | `[ ] Milestone 2+3` |
| CC9.2 — Vendor/partner risk | Build attestation (git_commit + build_id) in binary | `[ ] Milestone 4` |

---

## Minimum Implementation Order

The SOC 2 addendum specified this order (we keep it):

1. **Audit event schema + append-only log + hash chain** → Milestone 1
2. **Capability gate for verbs + deny events** → Milestone 2
3. **Policy snapshot hashing + change audit trail** → Milestone 4
4. **Domain lifecycle auditing (start/stop/exec/export)** → Milestone 2+
5. **Evidence export bundle** → Milestone 5

---

## What We Don't Claim

SOC 2 is an attestation about controls. We explicitly do not claim:

- Unhackability (no system is)
- Perfect availability (single-user, single-machine scope)
- Encrypted storage (encryption is a Milestone 6+ enhancement)
- Multi-factor authentication in v0 (hardware token support is future)

The scope is honest. An auditor knows exactly what they're looking at.

---

## Evidence Pack

`EXPORT_EVIDENCE()` produces a `.tar.gz` bundle containing:

```
evidence-<timestamp>/
├── manifest.json          -- SHA-256 hashes of all included files
├── audit/
│   └── audit-<start>-<end>.jsonl   -- audit log segment
├── policy/
│   ├── current-snapshot.json       -- active policy snapshot
│   └── snapshot-chain.json         -- hash chain of all snapshots
├── build/
│   └── attestation.json            -- build_id, git_commit, signing_key_id
└── integrity/
    └── chain-verify-result.json    -- result of VerifyChain() at export time
```

`manifest.json` is signed if `EMILY_SIGNING_KEY` is present in the environment.
