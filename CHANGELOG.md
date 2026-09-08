## 2026-09-08
- CI: add auto-release job matching PARENA's own pattern -- on every green push to main, auto-bumps a v{major}.{minor+1}.0 tag and publishes a real GitHub Release (regular, not prerelease) with a packaged emilyos-linux-amd64.tar.gz (binary + construct bundle). Previously only uploaded ephemeral, run-scoped CI artifacts with no persistent release (sess-20260905-0720-ec33e7c5)
- Pi image build: stage the expanded parenabusybox applet set (wc, head, yes, cat, sleep, env, on top of echo/basename/pwd/true/false) for aarch64, matching PARENA's own Phase 1 rollout. All 10 applets live-verified via qemu-aarch64-static against the real rootfs (sess-20260905-0720-ec33e7c5)
- Pi image build: stage PARENA-powered coreutils/shell (parenabusybox+parenash) cross-compiled for aarch64, available-not-default; found and fixed a real, previously-undiscovered musl-vs-glibc dynamic-linker ABI mismatch (via qemu-user-static execution testing, not just file(1) architecture checks) that also silently affected the existing emilyos binary -- fixed both via static linking, live-verified (sess-20260905-0720-ec33e7c5)
- Real rootfs (not synthetic) confirmed booting into OpenRC once /sbin/init exists -- root-lessly verified via a safe busybox --list check + hand-created symlink on a throwaway test copy. OpenRC caches service deps and finds /etc/init.d/emilyos. Confirms the privileged busybox --install trigger step is load-bearing and forecasts the real build should boot further once it runs. (sess-20260905-0720-ec33e7c5)
- Definitive root-less proof the Pi image boot pipeline works: QEMU screendump confirmed framebuffer console; a real boot.img + synthetic ext4 root partition test proved switch_root correctly finds/mounts /dev/mmcblk0p2 and transitions into it (screendump evidence at docs/pi-boot-test-switch-root-panic.png). (sess-20260905-0720-ec33e7c5)
- Attempted Phase 3 boot-test with root-lessly-bootstrapped qemu-system-aarch64 -M raspi3b. Real partial signal (hardware-probe activity confirms the kernel executes) but no visible console output yet -- likely a framebuffer-vs-UART console mismatch, not yet resolved. (sess-20260905-0720-ec33e7c5)
- Found and fixed a real boot-config gap by reading initramfs-rpi's own init script: Alpine's stock cmdline.txt has no root=, taking the diskless boot path instead of mounting the real ext4 root partition. build-pi-image-rootless.sh now writes a correct cmdline.txt; added missing root/fsck/localmount/swap/seedrng to sudo-queue/76's rc-update list. (sess-20260905-0720-ec33e7c5)
- Root-lessly bootstrapped an aarch64 cross-toolchain (apt-get download + dpkg-deb -x) closing the cgo cross-build gap for cmd/emilyos -> linux/arm64. Fixed two real wrinkles: target libc6:arm64 fetched from ports.ubuntu.com, absolute cross-gcc sysroot overridden via --sysroot. Live-verified: a real aarch64 ELF binary now builds end to end with zero root. (sess-20260905-0720-ec33e7c5)
- Split the Alpine/RPi image build into a root-less half (packaging/scripts/build-pi-image-rootless.sh, run and verified live) and a much smaller privileged half (sudo-queue/76). Found live: mke2fs -d needs root for Alpine's execute-only bbsuid; cmd/emilyos doesn't cross-compile to linux/arm64 with CGO_ENABLED=0 (fsaclmod is cgo-only). (sess-20260905-0720-ec33e7c5)

- Pivot NORTHSTAR_DISTRO.md to Alpine for the real Raspberry Pi target (resolves the base-mechanism and target-hardware open questions). Proved a root-less apk-tools-static rootfs bootstrap; named the real privilege boundary (chroot + qemu-user-static + loop-mount) and queued sudo-queue/76-build-emilyos-pi-image.sh for the privileged Phase 1 build. (sess-20260905-0720-ec33e7c5)

## 2026-08-28

- CI fix: enabled CGO for the emilyos build -- internal/fsaclmod (a real cgo-based PARENA mod added 2026-08-25) had been silently breaking the static build since. commit 2147f89. (sess-20260825-1938-f6bd411e)

## 2026-08-26

- README.md rewritten to reflect the real Go policy kernel (was a stale disconnected GUI v0.1 design doc); old content archived to docs/legacy-archive/gui-v0.1-design-capture.md (sess-20260825-1938-f6bd411e)

## 2026-08-25

- feat(fs): GRANT_FS/REVOKE_FS verbs -- real filesystem ACL grants as PARENA-authored mods (internal/fsacl + internal/fsaclmod, stdlib/emilyos/fsacl.prn), replacing sudo-queue/22 hand-run setfacl with capability-checked (cap.fs.grant/cap.fs.revoke, Admin-only), audited, standing-denylist-protected policy. Live round-trip tested through the real compiled PARENA mod, not just compile-checked. See EMILY/BACKLOG.md for the full writeup.

## 2026-08-20

- Added ada/posture, a Ravenscar-profile Ada port of the posture state machine's pure decision logic -- unverified, blocked on installing GNAT (queued in sudo-queue) (sess-20260813-2154-dda37e8b)

## 2026-06-25

- feat(ci): GitHub Actions CI workflow — test, static build (GOWORK=off), construct bundle

## 2026-06-21
- feat: S54-02 EXPORT_EVIDENCE verb audit + TestBundleManifestVerification (Milestone 5 complete) (Apple #2372)
- feat: S54-01 POLICY_ROLLBACK verb + TestSnapshotRollback (Milestone 4 complete) (Apple #2370)
- feat: S50-02 emilyos audit bundle — tar.gz SOC 2 evidence bundle (Apple #2353)
- feat: S50-01 policy snapshot store tests + emilyos about/snapshot commands (Apple #2349)
- feat: S48-03 emilyos audit history — posture transition log (Apple #2343)
- feat: S47-03 emilyos audit export — SOC 2 evidence bundle (Apple #2334)
- feat: S46-02 emilyos CLI (posture get/set, verb dispatch, audit tail/verify) (Apple #2322)

- feat: S46-01 RBAC + posture test coverage (policy 8 tests, posture 7 tests — all pass) (Apple #2320)

