// Package fsaclmod is EmilyOS's second real PARENA-authored mod (after
// PITVIPER's scrollmod, S192-01): GRANT_FS/REVOKE_FS wired through a
// PARENA-compiled C function rather than calling internal/fsacl
// directly from Go. Founder, real-time (2026-08-25): "do more work on
// the EmilyOS GRANT-FS REVOKE-FS with parena mods for arch linux."
//
// v0 scope, deliberately minimal, same shape as scrollmod: two PARENA
// functions (grant-fs/revoke-fs, in stdlib/emilyos/fsacl.prn) compiled
// to C and linked directly into this cgo package; EmilyOS's GRANT_FS/
// REVOKE_FS verb handlers call TriggerGrant/TriggerRevoke, which call
// the compiled mod's grant_fs/revoke_fs, which call back into
// emilyos_fsacl_grant/emilyos_fsacl_revoke (exported below) to run the
// real setfacl logic in internal/fsacl. See EMILY/BACKLOG.md's fsacl
// writeup for the full trail.
//
// fsacl_mod.c in this package directory is PARENA-generated (`parena
// build parena-src/fsacl_mod.prn -o fsacl_mod.c`) -- do not hand-edit
// it; regenerate from the .prn source instead.
package fsaclmod

/*
#cgo CFLAGS: -include ${SRCDIR}/fsaclmod_host.h
#include <stdlib.h>
extern int grant_fs(char *identity, char *path);
extern int revoke_fs(char *identity, char *path);
*/
import "C"

import (
	"unsafe"

	"emilyos/internal/fsacl"
)

// emilyos_fsacl_grant is called BY the PARENA-compiled mod (fsacl_mod.c's
// grant_fs), not the other way around -- the "host" half of the
// mod-surface ABI declared in fsaclmod_host.h. Returns 0 on success,
// nonzero on any error (denylist refusal or a real setfacl failure).
//
//export emilyos_fsacl_grant
func emilyos_fsacl_grant(identity, path *C.char) C.int {
	if err := fsacl.Grant(C.GoString(identity), C.GoString(path)); err != nil {
		return 1
	}
	return 0
}

//export emilyos_fsacl_revoke
func emilyos_fsacl_revoke(identity, path *C.char) C.int {
	if err := fsacl.Revoke(C.GoString(identity), C.GoString(path)); err != nil {
		return 1
	}
	return 0
}

// TriggerGrant is what EmilyOS's GRANT_FS verb handler calls. It enters
// the PARENA-compiled mod (grant_fs), which immediately calls back into
// emilyos_fsacl_grant above -- a real round-trip through compiled
// PARENA code, not a no-op passthrough. Returns nil on success, or the
// underlying error string wrapped for the caller (the PARENA boundary
// only carries an int status, so the specific Go error -- e.g. which
// denylist pattern matched -- is not recoverable across it; re-checking
// the denylist here separately below lets the verb handler still return
// a real, specific reason to the audit log rather than just "denied").
func TriggerGrant(identity, path string) error {
	cIdentity := C.CString(identity)
	defer C.free(unsafe.Pointer(cIdentity))
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	if status := C.grant_fs(cIdentity, cPath); status != 0 {
		// Real error already happened host-side (fsacl.Grant); re-run it
		// directly (same denylist-checked path) purely to recover the
		// specific error for the caller/audit log, not to re-attempt the
		// grant a second time -- if the first attempt got far enough to
		// actually run setfacl, this second call is idempotent (setfacl
		// re-applying the same ACL entry is a no-op, not a duplicate grant).
		return fsacl.Grant(identity, path)
	}
	return nil
}

// TriggerRevoke mirrors TriggerGrant for REVOKE_FS.
func TriggerRevoke(identity, path string) error {
	cIdentity := C.CString(identity)
	defer C.free(unsafe.Pointer(cIdentity))
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	if status := C.revoke_fs(cIdentity, cPath); status != 0 {
		return fsacl.Revoke(identity, path)
	}
	return nil
}
