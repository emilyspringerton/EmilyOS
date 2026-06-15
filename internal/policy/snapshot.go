// snapshot.go — policy versioning.
//
// Every policy change produces an immutable, hash-addressed JSON snapshot.
// Snapshots form a chain: each references its predecessor's ID.
// POLICY_ROLLBACK(snapshot_id) is an explicit verb that restores a prior snapshot.
package policy

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Snapshot is an immutable policy configuration record.
type Snapshot struct {
	SnapshotID     string         `json:"snapshot_id"`
	CreatedAt      time.Time      `json:"created_at"`
	ActorID        string         `json:"actor_id"`
	BuildID        string         `json:"build_id"`
	GitCommit      string         `json:"git_commit"`
	PrevSnapshotID string         `json:"prev_snapshot_id"`
	Roles          map[string]any `json:"roles"`
	Capabilities   map[string]any `json:"capabilities"`
	PosturePolicy  map[string]any `json:"posture_policy"`
	Hash           string         `json:"hash"`
}

// SnapshotStore manages policy snapshots on disk.
type SnapshotStore struct {
	dir string
}

// NewSnapshotStore creates a store backed by dir. The directory is created if absent.
func NewSnapshotStore(dir string) (*SnapshotStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("policy snapshots dir: %w", err)
	}
	return &SnapshotStore{dir: dir}, nil
}

// Write saves a snapshot to disk. The snapshot's Hash is computed and set.
func (s *SnapshotStore) Write(snap *Snapshot) error {
	snap.Hash = computeSnapshotHash(snap)
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	path := filepath.Join(s.dir, snap.SnapshotID+".json")
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
}

// Get returns the snapshot with the given ID.
func (s *SnapshotStore) Get(snapshotID string) (*Snapshot, error) {
	path := filepath.Join(s.dir, snapshotID+".json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("snapshot %q not found", snapshotID)
	}
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parse snapshot: %w", err)
	}
	return &snap, nil
}

// Latest returns the most recently created snapshot, or nil if none exist.
func (s *SnapshotStore) Latest() (*Snapshot, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return nil, nil
	}
	sort.Strings(names) // lexicographic; snapshot IDs embed timestamp
	latest := names[len(names)-1]
	return s.Get(latest[:len(latest)-5]) // strip .json
}

// NewSnapshotID returns a timestamp-based snapshot ID.
func NewSnapshotID() string {
	return "snap-" + time.Now().UTC().Format("20060102T150405Z")
}

func computeSnapshotHash(snap *Snapshot) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%s\n%s\n%s\n%s\n%s",
		snap.SnapshotID,
		snap.CreatedAt.UTC().Format(time.RFC3339),
		snap.ActorID,
		snap.BuildID,
		snap.GitCommit,
		snap.PrevSnapshotID,
	)
	roles, _ := json.Marshal(snap.Roles)
	caps, _ := json.Marshal(snap.Capabilities)
	posture, _ := json.Marshal(snap.PosturePolicy)
	h.Write(roles)
	h.Write(caps)
	h.Write(posture)
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}
