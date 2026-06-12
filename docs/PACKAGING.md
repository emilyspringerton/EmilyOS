# EmilyOS Packaging + Build System Decision

*Written: 2026-06-12 | Status: decided — Debian base, reproducible Go build*

---

## Decision: Debian Base

EmilyOS targets Ubuntu 22.04 LTS and Debian 12 ("Bookworm"). Debian base is the correct choice:

- CLAUDE.md specifies "Linux (Ubuntu 22.04 LTS or Debian 12 baseline)"
- systemd domain lifecycle management is Debian/Ubuntu native
- The SOC 2 audit trail (append-only JSONL) is a pure Go binary with no C dependencies
- `dpkg` + `apt` provides content-addressed package distribution (SHA-256 checksums in
  `Packages` index file, cryptographic signature via `Release.gpg`)

**Arch base: rejected.** Rolling-release model conflicts with SOC 2 change-management
requirements (audit trail must reflect controlled, versioned updates — not bleeding-edge
package churn).

---

## Reproducible Builds

EmilyOS is pure Go. Reproducible builds are achieved without Bazel:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -ldflags="-s -w -X main.Version=$(git describe --tags --always)" \
  -o dist/emilyos ./cmd/emilyos
```

Key properties:
- `-trimpath` removes local file paths from the binary
- `CGO_ENABLED=0` produces a fully static binary (no libc dependency)
- `go.sum` pins all module dependencies by SHA-256
- The same source + same `go.sum` + same Go toolchain version = byte-for-byte identical binary

For SOC 2 auditing: the binary hash is recorded in the audit log at install time via the
`INSTALL` verb. The hash is content-addressed: `sha256:<hash-of-binary>`.

---

## Package Repository Structure

```
packaging/
  debian/
    control         — package metadata (name, version, arch, depends, description)
    changelog       — Debian-format changelog (required by lintian)
    rules           — build rules (calls go build)
    copyright       — SPDX license file
    postinst        — post-install script (systemd enable, audit dir creation)
    prerm           — pre-removal script (systemd disable)
  scripts/
    build-deb.sh    — builds .deb from source using dpkg-buildpackage
    sign-release.sh — signs the apt Release file with GPG key
    publish.sh      — pushes to package repo (S3-backed apt mirror or self-hosted)
```

The `.deb` contains:
- `/usr/bin/emilyos` — the binary
- `/etc/emilyos/policy.json` — default policy (read-only, managed by policy verbs)
- `/var/lib/emilyos/` — runtime state directory (audit logs, posture.json)
- `/lib/systemd/system/emilyos.service` — systemd unit

---

## Content-Addressed Storage

Debian `Packages` index already SHA-256-hashes every `.deb`. For SOC 2 auditability,
we additionally:
1. At install, record the SHA-256 hash of the installed binary in the audit log as an
   `install` event with `object_ref: sha256:<hash>`.
2. On startup, `VerifyBinaryHash()` checks the running binary against the recorded hash.
   If mismatch → posture shifts to INCIDENT + audit event.

This provides tamper detection on the binary itself, not just the package.

---

## Build Targets

| Target | Command | Output |
|--------|---------|--------|
| Local binary | `make build` | `dist/emilyos` |
| Static binary | `make build-static` | `dist/emilyos` (CGO=0) |
| Debian package | `make deb` | `dist/emilyos_<version>_amd64.deb` |
| Docker image | `make docker` | `emilyos:<version>` |
| Verify chain | `make verify` | `go test ./...` |

---

## Bazel: Not Adopted

Bazel is overkill for a Go-only project. `go build` with `-trimpath` and pinned `go.sum`
already satisfies "reproducible builds, content-addressed storage." Bazel would add
build-system complexity without a proportional SOC 2 benefit.

If EmilyOS ever needs multi-language builds (C extensions, native libraries), revisit Bazel.
Until then, standard Go toolchain is the right choice.
