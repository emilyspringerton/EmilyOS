// Package audit implements EmilyOS's append-only, hash-chained audit log.
//
// Every meaningful event in EmilyOS is recorded here. The log is the source
// of truth for SOC 2 compliance. It is never modified after write — only
// appended to. Each event includes a SHA-256 hash over the previous event's
// hash, forming a tamper-evident chain.
//
// The first event in a log uses prev_hash = "sha256:genesis".
// VerifyChain walks all events and returns ErrTampered on any modification.
package audit

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const genesisHash = "sha256:genesis"

// Event is one record in the audit log. Matches SOC 2 §2.2 schema.
type Event struct {
	TS         time.Time      `json:"ts"`
	Seq        int64          `json:"seq"`
	ActorID    string         `json:"actor_id"`
	SessionID  string         `json:"session_id"`
	DeviceID   string         `json:"device_id"`
	Verb       string         `json:"verb"`
	ObjectRef  string         `json:"object_ref"`
	Decision   string         `json:"decision"` // allow | deny
	ReasonCode string         `json:"reason_code,omitempty"`
	Result     string         `json:"result"` // success | failure | error:...
	Meta       map[string]any `json:"meta,omitempty"`
	PrevHash   string         `json:"prev_hash"`
	Hash       string         `json:"hash"`
}

// Decision values.
const (
	Allow = "allow"
	Deny  = "deny"
)

// Result values.
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

// ErrTampered is returned by VerifyChain when an event has been modified.
type ErrTampered struct {
	EventSeq int64
	Reason   string
}

func (e ErrTampered) Error() string {
	return fmt.Sprintf("audit chain tampered at seq=%d: %s", e.EventSeq, e.Reason)
}

// Logger is a thread-safe, append-only, hash-chained audit log writer.
type Logger struct {
	mu       sync.Mutex
	f        *os.File
	seq      int64
	lastHash string
}

// Open opens or creates the audit log at path. If the file exists, the last
// event is read to initialize the hash chain and sequence counter.
func Open(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("audit: create dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o640)
	if err != nil {
		return nil, fmt.Errorf("audit: open log: %w", err)
	}
	l := &Logger{f: f, lastHash: genesisHash}

	// If the file has existing events, fast-scan to find the last one.
	events, err := ReadFile(path)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("audit: read existing events: %w", err)
	}
	if len(events) > 0 {
		last := events[len(events)-1]
		l.seq = last.Seq + 1
		l.lastHash = last.Hash
	}
	return l, nil
}

// Log writes a single audit event. The event's seq, ts, prev_hash, and hash
// are set by the logger; callers provide all other fields.
// If the write fails, the error is returned — the verb must not proceed.
func (l *Logger) Log(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	e.Seq = l.seq
	e.TS = time.Now().UTC()
	e.PrevHash = l.lastHash
	e.Hash = computeHash(e)

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("audit: marshal event: %w", err)
	}
	data = append(data, '\n')
	if _, err := l.f.Write(data); err != nil {
		return fmt.Errorf("audit: write event: %w", err)
	}
	if err := l.f.Sync(); err != nil {
		return fmt.Errorf("audit: sync: %w", err)
	}

	l.seq++
	l.lastHash = e.Hash
	return nil
}

// Allow is a convenience wrapper for logging a permitted verb execution.
func (l *Logger) Allow(actorID, sessionID, deviceID, verb, objectRef, result string, meta map[string]any) error {
	return l.Log(Event{
		ActorID:   actorID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		Verb:      verb,
		ObjectRef: objectRef,
		Decision:  Allow,
		Result:    result,
		Meta:      meta,
	})
}

// Deny is a convenience wrapper for logging a denied verb attempt.
func (l *Logger) Deny(actorID, sessionID, deviceID, verb, objectRef, reasonCode string, meta map[string]any) error {
	return l.Log(Event{
		ActorID:    actorID,
		SessionID:  sessionID,
		DeviceID:   deviceID,
		Verb:       verb,
		ObjectRef:  objectRef,
		Decision:   Deny,
		ReasonCode: reasonCode,
		Result:     ResultFailure,
		Meta:       meta,
	})
}

// Close flushes and closes the underlying file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.f.Sync(); err != nil {
		return err
	}
	return l.f.Close()
}

// ReadFile reads all events from a JSONL audit log file.
func ReadFile(path string) ([]Event, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var events []Event
	dec := json.NewDecoder(bytesReader(data))
	for dec.More() {
		var e Event
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("audit: decode event: %w", err)
		}
		events = append(events, e)
	}
	return events, nil
}

// VerifyChain reads all events from path and verifies the hash chain.
// Returns ErrTampered if any event has been modified.
func VerifyChain(path string) error {
	events, err := ReadFile(path)
	if err != nil {
		return err
	}
	return VerifyEvents(events)
}

// VerifyEvents verifies the hash chain of an already-loaded event slice.
func VerifyEvents(events []Event) error {
	if len(events) == 0 {
		return nil
	}
	// Verify first event
	if events[0].PrevHash != genesisHash {
		return ErrTampered{EventSeq: events[0].Seq, Reason: "first event prev_hash is not genesis"}
	}

	for i, e := range events {
		// Verify stored hash
		recomputed := computeHash(e)
		if recomputed != e.Hash {
			return ErrTampered{EventSeq: e.Seq, Reason: fmt.Sprintf("hash mismatch: stored=%s computed=%s", e.Hash, recomputed)}
		}
		// Verify linkage
		if i > 0 {
			if e.PrevHash != events[i-1].Hash {
				return ErrTampered{EventSeq: e.Seq, Reason: fmt.Sprintf("prev_hash mismatch: got=%s expected=%s", e.PrevHash, events[i-1].Hash)}
			}
		}
	}
	return nil
}

// computeHash computes the canonical hash for an event.
// The hash is over the immutable fields; Hash and PrevHash must already be set.
func computeHash(e Event) string {
	h := sha256.New()
	// Canonical order — must never change.
	fmt.Fprintf(h, "%d\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s",
		e.Seq,
		e.ActorID,
		e.SessionID,
		e.DeviceID,
		e.Verb,
		e.ObjectRef,
		e.Decision,
		e.ReasonCode,
		e.Result,
		e.PrevHash,
	)
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

// bytesReader wraps a byte slice in a no-copy reader for json.NewDecoder.
type bytesReaderT struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *bytesReaderT { return &bytesReaderT{data: data} }

func (r *bytesReaderT) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
