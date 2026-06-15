package audit_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"emilyos/internal/audit"
)

func TestAuditLog_WriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	logger, err := audit.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer logger.Close()

	for i := 0; i < 10; i++ {
		if err := logger.Allow("emily", "sess-1", "dev-1", "DOMAIN_EXEC", "domain:work", "success", nil); err != nil {
			t.Fatalf("log allow: %v", err)
		}
	}
	logger.Deny("emily", "sess-1", "dev-1", "DOMAIN_EXEC", "domain:game", "cap.net.denied", nil)

	events, err := audit.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(events) != 11 {
		t.Errorf("want 11 events, got %d", len(events))
	}

	// Verify chain is valid after honest writes
	if err := audit.VerifyChain(path); err != nil {
		t.Errorf("chain should be valid: %v", err)
	}

	// Check decision field
	if events[10].Decision != audit.Deny {
		t.Errorf("last event decision: got %q want %q", events[10].Decision, audit.Deny)
	}
	if events[10].ReasonCode != "cap.net.denied" {
		t.Errorf("reason_code: got %q want %q", events[10].ReasonCode, "cap.net.denied")
	}
}

func TestAuditLog_TamperDetected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	logger, err := audit.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for i := 0; i < 100; i++ {
		if err := logger.Allow("emily", "sess-1", "dev-1", "DOMAIN_EXEC", "domain:work", "success", nil); err != nil {
			t.Fatalf("log: %v", err)
		}
	}
	logger.Close()

	// Tamper with event 50: change the verb field (valid JSON, tampered content)
	data, _ := os.ReadFile(path)
	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
	if len(lines) != 100 {
		t.Fatalf("expected 100 lines, got %d", len(lines))
	}
	// Decode line 49 (seq=49), change verb, re-encode — the hash will no longer match
	var e50 map[string]any
	if err := json.Unmarshal(lines[49], &e50); err != nil {
		t.Fatalf("unmarshal event 50: %v", err)
	}
	e50["verb"] = "TAMPERED_VERB"
	lines[49], _ = json.Marshal(e50)
	out := bytes.Join(lines, []byte("\n"))
	out = append(out, '\n')
	os.WriteFile(path, out, 0o640)

	// Verify chain should detect tampering
	err = audit.VerifyChain(path)
	if err == nil {
		t.Fatal("expected tamper detection, got nil error")
	}
	var tampered audit.ErrTampered
	if !errors.As(err, &tampered) {
		t.Errorf("expected ErrTampered, got %T: %v", err, err)
	}
	t.Logf("tamper detected: %v", err)
}

func TestAuditLog_PersistsSeqAndHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	// First session: write 5 events
	l1, _ := audit.Open(path)
	for i := 0; i < 5; i++ {
		l1.Allow("emily", "sess-1", "dev-1", "ENTER", "/", "success", nil)
	}
	l1.Close()

	// Second session: open same file, write 5 more
	l2, err := audit.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	for i := 0; i < 5; i++ {
		l2.Allow("emily", "sess-2", "dev-1", "EXIT", "/", "success", nil)
	}
	l2.Close()

	events, _ := audit.ReadFile(path)
	if len(events) != 10 {
		t.Errorf("want 10 events across sessions, got %d", len(events))
	}
	if events[5].Seq != 5 {
		t.Errorf("seq continuity: want 5, got %d", events[5].Seq)
	}

	if err := audit.VerifyChain(path); err != nil {
		t.Errorf("chain should be valid across sessions: %v", err)
	}
}

