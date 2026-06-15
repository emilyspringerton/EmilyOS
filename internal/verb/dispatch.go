// Package verb implements the EmilyOS verb dispatcher.
//
// Every action in EmilyOS is a declared verb. The dispatcher:
//  1. Checks that the caller's role has the required capability
//  2. Checks that the current posture doesn't override that capability
//  3. Emits an audit event (allow or deny)
//  4. Executes the action handler (only on allow)
//
// No verb executes without an audit record. A verb without an audit record
// didn't happen.
package verb

import (
	"errors"
	"fmt"

	"emilyos/internal/audit"
	"emilyos/internal/policy"
	"emilyos/internal/posture"
)

// ErrDenied is returned when a verb is denied by capability check or posture.
type ErrDenied struct {
	Verb       string
	ReasonCode string
}

func (e ErrDenied) Error() string {
	return fmt.Sprintf("verb %s denied: %s", e.Verb, e.ReasonCode)
}

// Context carries the caller identity for a verb invocation.
type Context struct {
	ActorID   string
	SessionID string
	DeviceID  string
	Role      string
}

// Handler is a function that executes a verb's action.
type Handler func(ctx Context, objectRef string, meta map[string]any) error

// Dispatcher is the central verb dispatch engine.
type Dispatcher struct {
	log     *audit.Logger
	posture *posture.Machine
	verbs   map[string]verbDef
}

type verbDef struct {
	capability string
	handler    Handler
}

// New creates a Dispatcher.
func New(log *audit.Logger, pm *posture.Machine) *Dispatcher {
	return &Dispatcher{
		log:     log,
		posture: pm,
		verbs:   make(map[string]verbDef),
	}
}

// Register registers a verb with its required capability and handler.
func (d *Dispatcher) Register(verbName, capability string, handler Handler) {
	d.verbs[verbName] = verbDef{capability: capability, handler: handler}
}

// Dispatch executes a verb. Checks capability, checks posture, emits audit event.
// Returns ErrDenied if not permitted, or the handler's error on execution failure.
//
// This is the single entry point for all actions in EmilyOS.
// Callers must not bypass this — they must call Dispatch.
func (d *Dispatcher) Dispatch(ctx Context, verbName, objectRef string, meta map[string]any) error {
	def, ok := d.verbs[verbName]
	if !ok {
		// Unknown verb — deny and audit
		_ = d.log.Deny(ctx.ActorID, ctx.SessionID, ctx.DeviceID, verbName, objectRef,
			"verb.unknown", meta)
		return ErrDenied{Verb: verbName, ReasonCode: "verb.unknown"}
	}

	// Step 1: Check RBAC capability
	if !policy.HasCapability(ctx.Role, def.capability) {
		rc := fmt.Sprintf("rbac.%s.missing", def.capability)
		_ = d.log.Deny(ctx.ActorID, ctx.SessionID, ctx.DeviceID, verbName, objectRef, rc, meta)
		return ErrDenied{Verb: verbName, ReasonCode: rc}
	}

	// Step 2: Check posture override
	verdict := d.posture.CapabilityVerdict(def.capability)
	switch verdict {
	case posture.ForceOff:
		rc := fmt.Sprintf("posture.%s.denied", d.posture.Current())
		_ = d.log.Deny(ctx.ActorID, ctx.SessionID, ctx.DeviceID, verbName, objectRef, rc, meta)
		return ErrDenied{Verb: verbName, ReasonCode: rc}
	case posture.PinnedOnly:
		// In v0 we treat PinnedOnly as denied (pinned domain registry not yet implemented)
		rc := fmt.Sprintf("posture.%s.pinned_only", d.posture.Current())
		_ = d.log.Deny(ctx.ActorID, ctx.SessionID, ctx.DeviceID, verbName, objectRef, rc, meta)
		return ErrDenied{Verb: verbName, ReasonCode: rc}
	case posture.GameDomainOnly:
		// Only allowed if objectRef references the game domain
		if objectRef != "domain:game" {
			rc := fmt.Sprintf("posture.%s.game_domain_only", d.posture.Current())
			_ = d.log.Deny(ctx.ActorID, ctx.SessionID, ctx.DeviceID, verbName, objectRef, rc, meta)
			return ErrDenied{Verb: verbName, ReasonCode: rc}
		}
	case posture.ForceOn:
		// Posture grants this regardless of RBAC — proceed
	case posture.PassThrough:
		// Role-based capability was already checked above
	}

	// Step 3: Emit allow event, then execute
	if logErr := d.log.Allow(ctx.ActorID, ctx.SessionID, ctx.DeviceID, verbName, objectRef, "pending", meta); logErr != nil {
		// Audit write failure → deny execution (no audit = no action)
		return fmt.Errorf("verb %s: audit failure: %w", verbName, logErr)
	}

	// Step 4: Execute
	if err := def.handler(ctx, objectRef, meta); err != nil {
		// Log execution failure
		_ = d.log.Allow(ctx.ActorID, ctx.SessionID, ctx.DeviceID, verbName, objectRef,
			"error:"+err.Error(), meta)
		return err
	}

	// Log success (the "pending" event above records the intent; this records completion)
	_ = d.log.Allow(ctx.ActorID, ctx.SessionID, ctx.DeviceID, verbName, objectRef, audit.ResultSuccess, meta)
	return nil
}

// IsDenied returns true if the error is an ErrDenied.
func IsDenied(err error) bool {
	var d ErrDenied
	return errors.As(err, &d)
}
