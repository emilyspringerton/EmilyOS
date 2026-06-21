package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	. "emilyos/internal/policy"
)

func TestSnapshotWriteAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSnapshotStore(dir)
	if err != nil {
		t.Fatalf("NewSnapshotStore: %v", err)
	}

	id := NewSnapshotID()
	snap := &Snapshot{
		SnapshotID: id,
		ActorID:    "test-actor",
		Roles:      map[string]any{"operator": []string{"cap.exec"}},
	}
	if err := store.Write(snap); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if snap.Hash == "" {
		t.Error("expected Hash to be set after Write")
	}

	got, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ActorID != "test-actor" {
		t.Errorf("ActorID = %q, want test-actor", got.ActorID)
	}
	if got.Hash != snap.Hash {
		t.Errorf("Hash mismatch: stored=%s loaded=%s", snap.Hash, got.Hash)
	}
}

func TestSnapshotHashIsStable(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSnapshotStore(dir)

	snap := &Snapshot{
		SnapshotID: "snap-test",
		ActorID:    "alice",
		Roles:      map[string]any{"admin": []string{"cap.policy.write"}},
	}
	_ = store.Write(snap)
	h1 := snap.Hash

	// Writing a second identical snapshot should produce the same hash.
	snap2 := &Snapshot{
		SnapshotID: "snap-test",
		ActorID:    "alice",
		Roles:      map[string]any{"admin": []string{"cap.policy.write"}},
	}
	_ = store.Write(snap2)
	if h1 != snap2.Hash {
		t.Errorf("hash not stable: h1=%s h2=%s", h1, snap2.Hash)
	}
}

func TestSnapshotLatest(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSnapshotStore(dir)

	// No snapshots yet → Latest returns nil.
	latest, err := store.Latest()
	if err != nil || latest != nil {
		t.Errorf("Latest on empty dir: err=%v latest=%v", err, latest)
	}

	for _, id := range []string{"snap-20260101T000000Z", "snap-20260102T000000Z"} {
		s := &Snapshot{SnapshotID: id, ActorID: "actor"}
		_ = store.Write(s)
	}

	latest, err = store.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if latest == nil {
		t.Fatal("Latest returned nil")
	}
	if latest.SnapshotID != "snap-20260102T000000Z" {
		t.Errorf("Latest = %q, want snap-20260102T000000Z", latest.SnapshotID)
	}
}

func TestSnapshotGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSnapshotStore(dir)

	_, err := store.Get("does-not-exist")
	if err == nil {
		t.Error("expected error for missing snapshot, got nil")
	}
}

func TestSnapshotNewIDFormat(t *testing.T) {
	id := NewSnapshotID()
	if len(id) < 10 {
		t.Errorf("snapshot ID too short: %q", id)
	}
	// ID should start with "snap-"
	if id[:5] != "snap-" {
		t.Errorf("ID does not start with snap-: %q", id)
	}
}

// TestSnapshotRollback validates the 3-change → rollback-to-first scenario
// required by EmilyOS NORTHSTAR Milestone 4.
func TestSnapshotRollback(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSnapshotStore(dir)

	// Simulate 3 policy changes: write 3 snapshots with distinct role configs.
	snap1 := &Snapshot{
		SnapshotID: "snap-20260101T000001Z",
		ActorID:    "admin",
		Roles:      map[string]any{"operator": []string{"cap.exec", "cap.net"}},
	}
	snap2 := &Snapshot{
		SnapshotID:     "snap-20260101T000002Z",
		ActorID:        "admin",
		PrevSnapshotID: snap1.SnapshotID,
		Roles:          map[string]any{"operator": []string{"cap.exec"}}, // net removed
	}
	snap3 := &Snapshot{
		SnapshotID:     "snap-20260101T000003Z",
		ActorID:        "admin",
		PrevSnapshotID: snap2.SnapshotID,
		Roles:          map[string]any{"operator": []string{}}, // all caps removed
	}
	_ = store.Write(snap1)
	_ = store.Write(snap2)
	_ = store.Write(snap3)

	// Target: roll back to snap1.
	target, err := store.Get(snap1.SnapshotID)
	if err != nil {
		t.Fatalf("Get snap1: %v", err)
	}
	// Verify hash matches (rollback caller must verify before dispatching).
	computed := ComputeSnapshotHash(target)
	if target.Hash != computed {
		t.Errorf("snap1 hash mismatch: stored=%s computed=%s", target.Hash, computed)
	}
	// Check that snap1's role config is what we expect.
	ops, ok := target.Roles["operator"].([]any)
	if !ok {
		// json round-trip converts []string → []any
		t.Fatalf("snap1 operator roles: unexpected type %T", target.Roles["operator"])
	}
	if len(ops) != 2 {
		t.Errorf("snap1 operator role count: %d, want 2", len(ops))
	}
}

func TestSnapshotDirCreated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "snapshots")
	_, err := NewSnapshotStore(dir)
	if err != nil {
		t.Fatalf("NewSnapshotStore with non-existent dir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("dir not created: %v", err)
	}
}
