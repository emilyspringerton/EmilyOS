# EmilyOS — Audit Log Specification

**Version:** 0.1  
**Author:** Emily Prime  
**SOC 2 ref:** CC6.1, CC6.2, CC6.3, CC7.2

---

## Principle

Every meaningful event is loggable. Every loggable event is logged. The log is the source of truth.

No event is suppressed. A denied action is as important to record as an allowed one — arguably more so.

---

## Log Format

One JSON object per line (JSONL). Append-only. Never modified after write.

### Event Schema

```json
{
  "ts":          "2026-06-09T12:34:56.789012345Z",
  "seq":         42,
  "actor_id":    "emily-springerton",
  "session_id":  "sess-7f3a2c",
  "device_id":   "emily-prime-dev-01",
  "verb":        "DOMAIN_EXEC",
  "object_ref":  "domain:work/claude-code",
  "decision":    "allow",
  "reason_code": "",
  "result":      "success",
  "meta":        {},
  "prev_hash":   "sha256:e3b0c4429...",
  "hash":        "sha256:a665a4592..."
}
```

### Field Definitions

| Field | Type | Required | Description |
|---|---|---|---|
| `ts` | RFC3339Nano string | yes | Wall clock time (UTC) |
| `seq` | int64 | yes | Monotonically increasing sequence number |
| `actor_id` | string | yes | OperatorID or ServiceID. Never empty. |
| `session_id` | string | yes | Session identifier. Constant for session lifetime. |
| `device_id` | string | yes | Machine/device identifier |
| `verb` | string | yes | The verb that was invoked |
| `object_ref` | string | yes | What was acted upon (path, domain, resource) |
| `decision` | string | yes | `allow` or `deny` |
| `reason_code` | string | no | If `deny`: why (e.g. `cap.net.denied`, `posture.siege`) |
| `result` | string | yes | `success`, `failure`, or `error:...` |
| `meta` | object | no | Verb-specific structured metadata |
| `prev_hash` | string | yes | SHA-256 of the previous event's `hash` field |
| `hash` | string | yes | SHA-256 of `(seq + actor_id + session_id + device_id + verb + object_ref + decision + reason_code + result + prev_hash)` |

The first event in a log has `prev_hash = "sha256:genesis"`.

---

## Hash Chain

Each event's `hash` is computed over a canonical string:

```
hash = SHA-256( prev_hash + "\n" + seq + "\n" + actor_id + "\n" + ... )
```

The canonical ordering is fixed and must not change across versions (it is the audit chain). If the schema evolves, a new log is started with a `LOG_ROTATE` event linking the old log.

### Chain Verification

`VerifyChain(events []Event) error`:
1. For event 0: check `prev_hash == "sha256:genesis"`
2. For event N: recompute hash of event N-1, compare to `events[N].prev_hash`
3. Recompute `events[N].hash`, compare to stored `hash`
4. Any mismatch → `ErrTampered{EventSeq: N}`

---

## Event Categories

### UI / Session Events
- `SESSION_START` — login / session open
- `SESSION_END` — logout / session close
- `SESSION_LOCK` — idle timeout lock
- `SESSION_UNLOCK` — re-authentication
- `AUTH_FAILURE` — failed authentication attempt

### Verb Events
- `VERB_DISPATCHED` — any verb invoked (allow or deny)
- Specific verb events are emitted in addition for high-value actions

### Security Events
- `PRIVILEGE_CHANGE` — role assignment or revocation
- `POLICY_CHANGE` — RBAC or capability configuration change
- `POLICY_ROLLBACK` — explicit rollback to prior snapshot
- `POSTURE_CHANGE` — posture transition
- `INTEGRITY_FAILURE` — audit chain tamper detected
- `TAMPER_ATTEMPT` — attempt to write/modify audit log directly

### Domain Events
- `DOMAIN_START` — domain lifecycle start
- `DOMAIN_STOP` — domain lifecycle stop
- `DOMAIN_EXEC` — execution inside a domain
- `DOMAIN_SNAPSHOT` — domain state snapshotted
- `DOMAIN_EXPORT` — domain state exported

### SSH Events
- `SSH_HOST_ADD/REMOVE/EDIT` — host management
- `SSH_KEY_ADD/REMOVE/ROTATE` — key management
- `SSH_CONNECT_START/STOP` — connection lifecycle
- `SSH_TRUST_ACCEPT/DENY` — host key trust decision

### Audit Management Events
- `LOG_ROTATE` — audit log rotated (links to new log path)
- `EVIDENCE_EXPORT` — evidence bundle exported

---

## Log Storage

- Path: `var/audit.jsonl`
- Permissions: `0640`, owned by `emilyos` user, readable by `auditor` group
- No process except `emilyos` may write to this file
- The audit log directory (`var/`) should be on a dedicated filesystem with `nosuid,noexec` mount options in production

### Log Rotation

Log rotation is an explicit Admin verb (`LOG_ROTATE`), not an automatic process. When rotated:
1. A `LOG_ROTATE` event is written to the current log, referencing the new log path and its initial `prev_hash`
2. New log starts with a genesis event referencing the rotation event hash from the old log

This preserves the chain across rotation boundaries.

---

## Anti-Patterns (Never Do These)

- **Audit after execution**: capability check → audit → execute. Never execute then audit.
- **Swallow audit failures**: if the audit log write fails, the verb must fail. A verb without an audit record didn't happen.
- **Filter audit events**: every event is written. No "debug-only" events that can be turned off.
- **Log secrets**: `actor_id`, `session_id`, `device_id` are identifiers, not credentials. No passwords, no key material, no tokens.

---

## SOC 2 Mapping

| SOC 2 Criterion | Audit Mechanism |
|---|---|
| CC6.1 — Logical access | SESSION_START/END, AUTH_FAILURE |
| CC6.2 — Authentication | AUTH_FAILURE events, reason_code |
| CC6.3 — Role access | PRIVILEGE_CHANGE events |
| CC7.2 — System operations | DOMAIN_* events |
| CC8.1 — Change management | POLICY_CHANGE, POLICY_ROLLBACK |
| A1.1 — Availability monitoring | POSTURE_CHANGE (INCIDENT posture) |
| CC9.1 — Incident response | INCIDENT posture + EVIDENCE_EXPORT |
