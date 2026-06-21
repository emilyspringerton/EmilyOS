package posture_test

import (
	"os"
	"path/filepath"
	"testing"

	. "emilyos/internal/posture"
)

func tmpMachine(t *testing.T) *Machine {
	t.Helper()
	m, err := New(filepath.Join(t.TempDir(), "posture.json"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

func TestDefault_Normal(t *testing.T) {
	m := tmpMachine(t)
	if m.Current() != Normal {
		t.Errorf("default state = %q, want NORMAL", m.Current())
	}
}

func TestTransition_NormalToSiege(t *testing.T) {
	m := tmpMachine(t)
	oldState, err := m.Transition(Siege, "emily", "s1")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if oldState != Normal {
		t.Errorf("oldState = %q, want NORMAL", oldState)
	}
	if m.Current() != Siege {
		t.Errorf("current = %q, want SIEGE", m.Current())
	}
}

func TestTransition_InvalidReturnsError(t *testing.T) {
	m := tmpMachine(t)
	// SIEGE → INCIDENT is not in the transitions table.
	m.Transition(Siege, "emily", "s1")
	_, err := m.Transition(Incident, "emily", "s1")
	if err == nil {
		t.Fatal("SIEGE → INCIDENT should be an invalid transition")
	}
}

func TestTransition_Persisted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "posture.json")

	m1, _ := New(path)
	if _, err := m1.Transition(Siege, "emily", "s1"); err != nil {
		t.Fatalf("transition: %v", err)
	}

	// Reload from disk.
	m2, err := New(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if m2.Current() != Siege {
		t.Errorf("after reload: state = %q, want SIEGE", m2.Current())
	}
}

func TestCapabilityVerdict_SiegeDeniesNet(t *testing.T) {
	m := tmpMachine(t)
	m.Transition(Siege, "emily", "s1")
	v := m.CapabilityVerdict("cap.net")
	if v != ForceOff {
		t.Errorf("SIEGE cap.net = %v, want ForceOff (%d)", v, ForceOff)
	}
}

func TestCapabilityVerdict_NormalPassThrough(t *testing.T) {
	m := tmpMachine(t)
	v := m.CapabilityVerdict("cap.exec")
	if v != PassThrough {
		t.Errorf("NORMAL cap.exec = %v, want PassThrough (%d)", v, PassThrough)
	}
}

func TestCapabilityVerdict_IncidentForceOnExport(t *testing.T) {
	m := tmpMachine(t)
	// NORMAL → INCIDENT is a valid transition.
	m.Transition(Incident, "emily", "s1")
	v := m.CapabilityVerdict("cap.export")
	if v != ForceOn {
		t.Errorf("INCIDENT cap.export = %v, want ForceOn (%d)", v, ForceOn)
	}
}

func TestPostureFile_CreatedOnTransition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "posture.json")
	m, _ := New(path)
	m.Transition(Siege, "emily", "s1")

	if _, err := os.Stat(path); err != nil {
		t.Errorf("posture.json should exist after transition: %v", err)
	}
}
