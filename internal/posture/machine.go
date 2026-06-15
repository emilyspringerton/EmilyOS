// Package posture implements the EmilyOS posture state machine.
//
// Posture is the current operating mode of the system. It overrides RBAC for
// certain capabilities — posture is physics, RBAC is policy.
//
// States: NORMAL | SIEGE | MERCY | INCIDENT | GAME | EXITED
//
// Posture persists across restarts via var/posture.json.
package posture

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// State names.
const (
	Normal   = "NORMAL"
	Siege    = "SIEGE"
	Mercy    = "MERCY"
	Incident = "INCIDENT"
	Game     = "GAME"
	Exited   = "EXITED"
)

// ErrInvalidTransition is returned when a transition is not permitted.
type ErrInvalidTransition struct {
	From, To string
}

func (e ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid posture transition: %s → %s", e.From, e.To)
}

// CapabilityVerdict describes how a posture affects a capability.
type CapabilityVerdict int

const (
	PassThrough CapabilityVerdict = iota // posture doesn't override; fall through to RBAC
	ForceOff                             // posture hard-denies this capability
	ForceOn                              // posture grants this capability regardless of role
	PinnedOnly                           // capability allowed only for pinned domains
	GameDomainOnly                       // capability allowed only for the game domain
)

// postureCapOverrides defines how each posture overrides specific capabilities.
// A capability not listed here has PassThrough verdict.
var postureCapOverrides = map[string]map[string]CapabilityVerdict{
	Normal: {}, // no overrides
	Siege: {
		"cap.net":         ForceOff,
		"cap.domain.start": PinnedOnly,
	},
	Mercy: {
		"cap.exec":         PinnedOnly,
		"cap.domain.start": PinnedOnly,
	},
	Incident: {
		"cap.net":         ForceOff,
		"cap.exec":        ForceOff,
		"cap.domain.start": ForceOff,
		"cap.export":      ForceOn, // incident always allows evidence export
	},
	Game: {
		"cap.net":         ForceOff,
		"cap.domain.start": GameDomainOnly,
		"cap.exec":        GameDomainOnly,
	},
	Exited: {}, // no capabilities allowed in exited state (handled separately)
}

// transitions defines which (from, to) pairs are permitted.
var transitions = map[string]map[string]bool{
	Normal:   {Siege: true, Mercy: true, Game: true, Incident: true, Exited: true},
	Siege:    {Normal: true, Game: true, Exited: true},
	Mercy:    {Normal: true, Siege: true, Game: true, Exited: true},
	Incident: {Normal: true, Exited: true},
	Game:     {Normal: true, Exited: true}, // GAME is a toggle; GAME→NORMAL is the exit
}

// persistence is the on-disk format for var/posture.json.
type persistence struct {
	Posture   string    `json:"posture"`
	EnteredAt time.Time `json:"entered_at"`
	EnteredBy string    `json:"entered_by"`
	SessionID string    `json:"session_id"`
}

// Machine is the posture state machine.
type Machine struct {
	mu      sync.RWMutex
	current string
	path    string // path to var/posture.json
}

// New creates a Machine. If path exists, posture is loaded from it; otherwise NORMAL.
func New(path string) (*Machine, error) {
	m := &Machine{path: path, current: Normal}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("posture: read state: %w", err)
	}
	var p persistence
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("posture: parse state: %w", err)
	}
	if isValidState(p.Posture) {
		m.current = p.Posture
	}
	return m, nil
}

// Current returns the current posture state.
func (m *Machine) Current() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Transition attempts to move to the target state. Returns the old state and
// an error if the transition is not permitted.
func (m *Machine) Transition(to, actorID, sessionID string) (oldState string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	old := m.current
	if m.current == Exited {
		return old, fmt.Errorf("system exited")
	}

	// GAME is a toggle: GAME while in GAME returns to NORMAL.
	if m.current == Game && to == Game {
		to = Normal
	}

	if !transitions[m.current][to] {
		return old, ErrInvalidTransition{From: m.current, To: to}
	}

	m.current = to
	if err := m.persist(actorID, sessionID); err != nil {
		// Revert on persist failure
		m.current = old
		return old, fmt.Errorf("posture: persist: %w", err)
	}
	return old, nil
}

// CapabilityVerdict returns how the current posture treats the given capability.
func (m *Machine) CapabilityVerdict(cap string) CapabilityVerdict {
	m.mu.RLock()
	defer m.mu.RUnlock()
	overrides, ok := postureCapOverrides[m.current]
	if !ok {
		return PassThrough
	}
	v, ok := overrides[cap]
	if !ok {
		return PassThrough
	}
	return v
}

func (m *Machine) persist(actorID, sessionID string) error {
	p := persistence{
		Posture:   m.current,
		EnteredAt: time.Now().UTC(),
		EnteredBy: actorID,
		SessionID: sessionID,
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0o640)
}

func isValidState(s string) bool {
	switch s {
	case Normal, Siege, Mercy, Incident, Game, Exited:
		return true
	default:
		return false
	}
}
