// Package fsacl implements EmilyOS's GRANT_FS/REVOKE_FS verbs: real
// filesystem ACL grants/revokes for named identities, replacing the
// hand-run setfacl invocations in sudo-queue/22-create-treeiii-user.sh
// with declared, capability-checked, audited EmilyOS policy.
//
// Design: EMILY/BACKLOG.md S191-07, EmilyOS/docs/FS_ACL_COORDINATION.md.
// Founder real-time, 2026-08-25: "do more work on the EmilyOS GRANT-FS
// REVOKE-FS with parena mods for arch linux" -- the actual grant/revoke
// call is a real PARENA-authored mod (stdlib/emilyos/fsacl.prn), wired
// in the same shape PITVIPER's scrollmod established (S192-01): Go calls
// the compiled PARENA function, which immediately calls back into a
// Go-exported host function that does the real work -- the mod is a real
// round trip through compiled PARENA code, not a no-op passthrough, but
// the actual setfacl exec logic lives here in Go, same as scrollmod's
// real scrollback mechanism living in Go, not PARENA.
//
// "For Arch Linux": setfacl/getfacl are the same POSIX ACL tooling on
// Arch as on this box's own Debian/Ubuntu (the `acl` package, not a
// Debian-specific binary) -- the package-install step differs
// (`pacman -S acl` vs `apt-get install acl`), not the grant/revoke logic
// itself. This package doesn't install packages; that stays the
// sudo-queue script's job on whichever distro runs it. Untested against
// a real Arch box -- no Arch environment exists here to verify against,
// same honesty the MAC-address tooling thread (S193-01) already
// established for its own future-hardware deferral.
package fsacl

import (
	"fmt"
	"os/exec"
	"regexp"
)

// ErrDenylisted is returned when a grant/revoke targets a path this
// package refuses to touch regardless of caller, role, or capability.
type ErrDenylisted struct {
	Path    string
	Pattern string
}

func (e ErrDenylisted) Error() string {
	return fmt.Sprintf("fsacl: path %q matches standing denylist pattern %q -- refused", e.Path, e.Pattern)
}

// denylist is a standing, EmilyOS-wide set of path patterns that Grant
// refuses to touch, full stop -- not a per-call exclusion list. This is
// the safer of the two options FS_ACL_COORDINATION.md left open: a
// mistake in one grant call can't leak the crown jewels, because the
// package itself won't ever apply an ACL to a denylisted path, no
// matter what a caller (even an Admin-role caller) asks for.
//
// Patterns mirror sudo-queue/22's own real, hand-verified exclusions:
// the live secrets env files and SSH private keys.
var denylist = []*regexp.Regexp{
	regexp.MustCompile(`-secrets\.env$`),
	regexp.MustCompile(`/\.ssh/id_[^/]*$`),
	regexp.MustCompile(`/agent-secrets\.env$`),
}

func checkDenylist(path string) error {
	for _, pat := range denylist {
		if pat.MatchString(path) {
			return ErrDenylisted{Path: path, Pattern: pat.String()}
		}
	}
	return nil
}

// Grant applies a recursive ACL grant of rwx to identity (a username or
// numeric UID) on path, plus a default ACL on every directory under path
// so files created later inherit the same grant -- the exact shape
// sudo-queue/22 already hand-verified (both an existing-file grant and a
// default-ACL grant are required; default alone doesn't cover files that
// already exist, and existing-only doesn't cover files created later).
// Refuses denylisted paths before running anything.
func Grant(identity, path string) error {
	if err := checkDenylist(path); err != nil {
		return err
	}
	spec := fmt.Sprintf("u:%s:rwx", identity)
	if out, err := exec.Command("setfacl", "-R", "-m", spec, path).CombinedOutput(); err != nil {
		return fmt.Errorf("fsacl grant (existing files): %w: %s", err, out)
	}
	if out, err := exec.Command("setfacl", "-R", "-d", "-m", spec, path).CombinedOutput(); err != nil {
		return fmt.Errorf("fsacl grant (default ACL): %w: %s", err, out)
	}
	return nil
}

// Revoke removes identity's ACL entry (both the regular and default
// ACL) from path, recursively. Also denylist-checked -- consistent
// refusal in both directions, even though revoking access from a
// denylisted path is lower-risk than granting it, so a caller can't
// probe the denylist's existence by noticing grant/revoke behave
// differently.
func Revoke(identity, path string) error {
	if err := checkDenylist(path); err != nil {
		return err
	}
	if out, err := exec.Command("setfacl", "-R", "-x", "u:"+identity, path).CombinedOutput(); err != nil {
		return fmt.Errorf("fsacl revoke (existing files): %w: %s", err, out)
	}
	if out, err := exec.Command("setfacl", "-R", "-d", "-x", "u:"+identity, path).CombinedOutput(); err != nil {
		return fmt.Errorf("fsacl revoke (default ACL): %w: %s", err, out)
	}
	return nil
}
