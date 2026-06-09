# EmilyOS — Posture State Machine

**Version:** 0.1  
**Author:** Emily Prime  

---

## States

```
         GAME ←──────────────────┐
           │                     │
   SIEGE ←─┼──── NORMAL ────→ MERCY
           │       │  ↑           │
           └───────┼──────── INCIDENT
                   ↓
                EXITED
```

| State | Meaning | From Legacy |
|---|---|---|
| `NORMAL` | Default operating mode. All role capabilities apply. | "STATE_OPEN" |
| `SIEGE` | Hostile external environment. Network hard-off. | Spec §"Siege" |
| `MERCY` | Operator relief mode. Demands reduced. Only pinned domains. | Spec §"Mercy" |
| `INCIDENT` | Active incident. No exec. Preserve and export only. | SOC 2 §6.2 |
| `GAME` | Bounded offline space. Offline by construction. | Doc 10 |
| `EXITED` | Clean shutdown. Final state. | VERB_EXIT |

---

## Transitions (normative)

| From | Verb | To | Capability Required |
|---|---|---|---|
| NORMAL | SIEGE | SIEGE | `cap.posture.set` |
| NORMAL | MERCY | MERCY | `cap.posture.set` |
| NORMAL | GAME | GAME | `cap.posture.set` |
| NORMAL | INCIDENT | INCIDENT | `cap.posture.set` (Admin only) |
| NORMAL | EXIT | EXITED | `cap.session.open` |
| SIEGE | RESUME | NORMAL | `cap.posture.set` |
| SIEGE | GAME | GAME | `cap.posture.set` |
| SIEGE | EXIT | EXITED | `cap.session.open` |
| MERCY | RESUME | NORMAL | `cap.posture.set` |
| MERCY | SIEGE | SIEGE | `cap.posture.set` |
| MERCY | GAME | GAME | `cap.posture.set` |
| MERCY | EXIT | EXITED | `cap.session.open` |
| INCIDENT | RESUME | NORMAL | `cap.posture.set` (Admin only) |
| INCIDENT | EXIT | EXITED | `cap.session.open` |
| GAME | GAME | NORMAL | `cap.posture.set` (toggle) |
| GAME | EXIT | EXITED | `cap.session.open` |

**Rule:** Any undefined transition is DENIED and audited.

---

## Capability Overrides Per Posture

Posture overrides are applied *after* RBAC. They cannot be bypassed by any role.

```
NORMAL:   No overrides — role capabilities apply directly
SIEGE:    cap.net         = FORCE_OFF
          cap.domain.start = pinned-only
MERCY:    cap.exec        = pinned-only
          cap.domain.start = pinned-only
          (network remains role-governed)
INCIDENT: cap.net         = FORCE_OFF
          cap.exec        = FORCE_OFF
          cap.domain.start = FORCE_OFF
          cap.export      = ALLOW (override even if role lacks it)
GAME:     cap.net         = FORCE_OFF
          cap.domain.start = game-domain-only
          cap.exec        = game-domain-only
```

---

## Posture Persistence

Current posture is stored at `var/posture.json`:

```json
{
  "posture": "SIEGE",
  "entered_at": "2026-06-09T14:22:00Z",
  "entered_by": "emily-springerton",
  "session_id": "sess-abc123"
}
```

On startup, the posture is loaded from this file. If the file is absent, posture is `NORMAL`.

If the system crashes during `INCIDENT`, it comes back in `INCIDENT` — not `NORMAL`. This is intentional: incidents don't auto-resolve.

---

## Transition Rules (from legacy spec, formalized)

**G0 (from Doc 10) — GAME is offline by construction:**
While `GAME` is active, `cap.net` is treated as `FORCE_OFF` regardless of role or session.

**S0 — SIEGE means hostile outside:**
SIEGE is entered when the operator declares the external environment hostile. Network goes off. No exceptions. The only way out is explicit `RESUME`.

**I0 — INCIDENT preserves and exports:**
In INCIDENT, the only allowed mutations are `EXPORT` and posture transition. No new execution. No new domains. Preserve what we have.

**M0 — MERCY is not punishment:**
MERCY is operator relief. The system reduces its own demands. Pinned domains continue. Unpinned domains are paused. `cap.net` is not changed — only exec scope is reduced.

---

## Audit Events for Posture

Every posture transition emits:

```json
{
  "verb": "POSTURE_CHANGE",
  "object_ref": "posture",
  "decision": "allow",
  "result": "success",
  "meta": {
    "old_posture": "NORMAL",
    "new_posture": "SIEGE",
    "reason": "operator-declared"
  }
}
```
