package fsacl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testIdentity is a numeric UID with no matching real username -- setfacl
// accepts this directly (verified live against this box before writing
// this test), so real grant/revoke round-trips don't need a real second
// Linux account to exist.
const testIdentity = "9999"

func requireSetfacl(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("setfacl"); err != nil {
		t.Skip("setfacl not installed, skipping live ACL test")
	}
}

func TestGrantThenRevokeRoundTrip(t *testing.T) {
	requireSetfacl(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	if err := Grant(testIdentity, dir); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	out, err := exec.Command("getfacl", f).CombinedOutput()
	if err != nil {
		t.Fatalf("getfacl after grant: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "user:9999:rwx") {
		t.Errorf("getfacl after grant missing user:9999:rwx entry, got:\n%s", out)
	}

	if err := Revoke(testIdentity, dir); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	out, err = exec.Command("getfacl", f).CombinedOutput()
	if err != nil {
		t.Fatalf("getfacl after revoke: %v: %s", err, out)
	}
	if strings.Contains(string(out), "user:9999:") {
		t.Errorf("getfacl after revoke still has a user:9999 entry, got:\n%s", out)
	}
}

func TestGrantDefaultACLCoversFutureFiles(t *testing.T) {
	requireSetfacl(t)
	dir := t.TempDir()

	if err := Grant(testIdentity, dir); err != nil {
		t.Fatalf("Grant: %v", err)
	}

	// A file created AFTER the grant should still carry the ACL entry,
	// via the default-ACL half of Grant -- the actual "permissions cant
	// be fucked up when fatbaby comes back" fix sudo-queue/22 established
	// by hand; this test proves fsacl.Grant reproduces that behavior.
	laterFile := filepath.Join(dir, "created-after-grant.txt")
	if err := os.WriteFile(laterFile, []byte("y"), 0o644); err != nil {
		t.Fatalf("write later file: %v", err)
	}
	out, err := exec.Command("getfacl", laterFile).CombinedOutput()
	if err != nil {
		t.Fatalf("getfacl: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "user:9999:rwx") {
		t.Errorf("file created after Grant is missing the inherited default-ACL entry, got:\n%s", out)
	}
}

func TestGrantRefusesDenylistedSecretsFile(t *testing.T) {
	requireSetfacl(t)
	dir := t.TempDir()
	secrets := filepath.Join(dir, "emily-secrets.env")
	if err := os.WriteFile(secrets, []byte("KEY=x"), 0o600); err != nil {
		t.Fatalf("write secrets file: %v", err)
	}

	err := Grant(testIdentity, secrets)
	if err == nil {
		t.Fatal("Grant on a *-secrets.env path succeeded, want ErrDenylisted")
	}
	if _, ok := err.(ErrDenylisted); !ok {
		t.Errorf("Grant on secrets file returned %T, want ErrDenylisted: %v", err, err)
	}

	out, gerr := exec.Command("getfacl", secrets).CombinedOutput()
	if gerr != nil {
		t.Fatalf("getfacl: %v: %s", gerr, out)
	}
	if strings.Contains(string(out), "user:9999:") {
		t.Error("denylisted secrets file has a user:9999 ACL entry -- denylist did not actually prevent the grant")
	}
}

func TestGrantRefusesDenylistedSSHKey(t *testing.T) {
	requireSetfacl(t)
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}
	key := filepath.Join(sshDir, "id_ed25519")
	if err := os.WriteFile(key, []byte("fake-key"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	err := Grant(testIdentity, key)
	if err == nil {
		t.Fatal("Grant on an id_ed25519 path succeeded, want ErrDenylisted")
	}
	if _, ok := err.(ErrDenylisted); !ok {
		t.Errorf("Grant on ssh key returned %T, want ErrDenylisted: %v", err, err)
	}
}
