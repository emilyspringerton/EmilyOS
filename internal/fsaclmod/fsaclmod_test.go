package fsaclmod

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTriggerGrantRoundTrip actually executes the full
// Go -> cgo -> PARENA-compiled-C -> cgo-export -> Go-fsacl.Grant path,
// same live-round-trip bar PITVIPER's scrollmod test set (S192-01) --
// not just a compile check.
func TestTriggerGrantRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("setfacl"); err != nil {
		t.Skip("setfacl not installed, skipping live ACL test")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	if err := TriggerGrant("9999", dir); err != nil {
		t.Fatalf("TriggerGrant: %v", err)
	}
	out, err := exec.Command("getfacl", f).CombinedOutput()
	if err != nil {
		t.Fatalf("getfacl: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "user:9999:rwx") {
		t.Errorf("getfacl after TriggerGrant missing user:9999:rwx, got:\n%s", out)
	}

	if err := TriggerRevoke("9999", dir); err != nil {
		t.Fatalf("TriggerRevoke: %v", err)
	}
	out, err = exec.Command("getfacl", f).CombinedOutput()
	if err != nil {
		t.Fatalf("getfacl: %v: %s", err, out)
	}
	if strings.Contains(string(out), "user:9999:") {
		t.Errorf("getfacl after TriggerRevoke still has user:9999 entry, got:\n%s", out)
	}
}

// TestTriggerGrantDenylistPropagatesThroughParena confirms the denylist
// refusal (Go-side, internal/fsacl) survives the round trip through the
// PARENA-compiled mod and back -- not bypassed by going through the mod
// boundary.
func TestTriggerGrantDenylistPropagatesThroughParena(t *testing.T) {
	if _, err := exec.LookPath("setfacl"); err != nil {
		t.Skip("setfacl not installed, skipping live ACL test")
	}
	dir := t.TempDir()
	secrets := filepath.Join(dir, "agent-secrets.env")
	if err := os.WriteFile(secrets, []byte("KEY=x"), 0o600); err != nil {
		t.Fatalf("write secrets file: %v", err)
	}

	if err := TriggerGrant("9999", secrets); err == nil {
		t.Fatal("TriggerGrant on a denylisted path succeeded, want an error")
	}

	out, gerr := exec.Command("getfacl", secrets).CombinedOutput()
	if gerr != nil {
		t.Fatalf("getfacl: %v: %s", gerr, out)
	}
	if strings.Contains(string(out), "user:9999:") {
		t.Error("denylisted secrets file has a user:9999 ACL entry after going through the PARENA mod -- denylist bypassed")
	}
}
