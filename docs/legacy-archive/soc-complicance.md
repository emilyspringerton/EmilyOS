EmilyOS 2 — SOC 2 Compliance Addendum v0.1
 
emilyspringerton edited this page on Feb 3 · 1 revision
EmilyOS 2 — SOC 2 Compliance Addendum v0.1 0) Compliance posture

Default SOC 2 scope: Security (Common Criteria) only.

Optional modules (enable per contract): Availability, Confidentiality, Privacy, Processing Integrity.

This addendum defines what the product must do so an auditor can test it.

Identity, Auth, and Access Control 1.1 Operator identity is explicit
Every session has a single OperatorID (human) or ServiceID (automation).

No anonymous actions. If the system is in “demo/no-auth mode,” it must show a persistent banner: UNTRUSTED MODE.

1.2 Role-based access control (RBAC)

Minimum roles:

Operator: normal use (open dirs, run allowed apps, rename within permissions)

Admin: policy changes, domain templates, export policy

Auditor: read-only access to logs + configuration snapshots

1.3 Least privilege by default

UI verbs are capability-gated:

OPEN, RUN, EXPORT, RENAME, DELETE, POLICY_CHANGE, DOMAIN_EXEC, etc.

If a tile/action lacks capability → action is Denied (see §4.3 feedback).

Audit Logging as a First-Class Subsystem 2.1 “Every meaningful event is loggable”
Log the following as immutable events:

UI Events

focus/select tile

fast double click activation (ACTIVATE)

slow double click rename enter/commit/cancel (EDIT_LABEL)

pane split/resize/close

domain start/stop/exec/snapshot/export

Security Events

login/logout

auth failures

privilege escalation / role change

policy edits

integrity/attestation failures

audit log tamper attempts

2.2 Audit event schema (fixed)

Each event must record:

ts (monotonic + wall clock if available)

actor_id

session_id

device_id

verb

object_ref (tile/path/hash)

decision (allow/deny)

reason_code (if deny)

result (success/failure + errno)

2.3 Log properties

Append-only (WORM semantics).

Tamper-evident: hash-chain events (each entry includes prev_hash). (This is the simplest story auditors understand.)

Exportable as an “audit artifact” using the explicit EXPORT verb.

Change Management & Configuration Control 3.1 Policy changes are versioned
Every config/policy change produces:

a new immutable policy_snapshot (hash-addressed)

an audit event linking old → new

Rollback is allowed only as an explicit verb: POLICY_ROLLBACK(snapshot_hash) and is audited.

3.2 Build provenance (for the GUI and kernel)

Each build embeds:

build_id

git_commit

signing_key_id (if signed)

UI has a read-only “About / Attestation” pane to show these.

Operational Security Controls that touch the GUI 4.1 No single-click helps SOC 2
Your “double click = declared intent” rule becomes a control:

accidental actions are reduced

auditors can test “actions require deliberate input”

4.2 Deny behavior is quiet + auditable

When denied:

minimal visual deny flash (no modal)

must emit an audit event with decision=deny and reason_code

4.3 Session timeout / lock

Idle timeout locks the session (policy-defined).

Unlock requires re-auth (or hardware token if you support it later).

Lock/unlock events are audited.

Isolation & Legacy Compatibility Domain (SOC 2-friendly story) 5.1 Domains are “systems in scope”
EmilyOS UI + policy kernel is the control plane.

Linux/LXC is the isolation substrate.

Debian-in-LXC is a compatibility domain.

5.2 Domain defaults (Security baseline)

Default template requirements:

no network (unless explicitly granted)

read-only rootfs by default

minimal mounts (no host home)

explicit domain.exec allowlist

All domain lifecycle events are audited.

Monitoring, Incident Response, and Evidence 6.1 Evidence pack (one-click via EXPORT, still intent-declared)
EXPORT_EVIDENCE() creates a bundle:

last N days audit log segment

current policy snapshot + hash

domain templates + hashes

build attestation info

(optional) integrity check results

6.2 Incident mode

Add a posture:

INCIDENT clamps:

network off

exec allowlist shrinks

exports allowed

preserve logs All of this is auditable.

What we don’t claim (keeps you honest in SOC 2)
SOC 2 is an attestation about controls, not a magical “certification.”

We do not claim “unhackable”; we claim:

deliberate interaction

least privilege

isolation boundaries

tamper-evident audit trail

controlled change management

Minimal implementation order (so this ships)

Audit event schema + append-only log + hash-chain

Capability gate for verbs + deny events

Policy snapshot hashing + policy change audit trail

Domain lifecycle auditing (start/stop/exec/export)

Evidence export bundle
