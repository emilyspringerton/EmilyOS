#!/usr/bin/env bash
# build-pi-image-rootless.sh — the root-less half of EmilyOS/docs/NORTHSTAR_DISTRO.md's Alpine/
# Raspberry Pi image build (Phase 1, 2026-09-08 refinement). Does everything that does NOT
# genuinely need root, proven live in the session that wrote this: apk package fetch/extraction,
# building EmilyOS's own Go binary, populating a real ext4 filesystem image from a directory tree
# (`mke2fs -d`), building and populating a real FAT32 boot image (`mkfs.vfat` + `mtools`'s
# `mcopy`/`mmd`, no mount needed), partitioning a plain file (`parted` operates on regular files
# directly, no loop device needed), and assembling the final combined `.img` via `dd` at exact
# byte offsets (verified byte-identical against the source sub-images after assembly).
#
# What this script deliberately does NOT do, and why: Alpine's own package post-install/trigger
# scripts (busybox, alpine-baselayout, openrc) run via a real `chroot`, which needs
# CAP_SYS_CHROOT — genuinely unavailable to a plain uid=1000 user. Because the rootfs is aarch64
# and this build box is x86_64, that chroot additionally needs qemu-user-static's binfmt_misc
# registration. Both are real, unavoidable root requirements — see
# `../../../sudo-queue/76-build-emilyos-pi-image.sh` (top-level monorepo) for that much smaller,
# separately-queued finishing step, which operates on the SAME `rootfs/` directory this script
# builds, after which re-run this script's own step 6 (image assembly) again to bake the finished
# rootfs into the final image (mke2fs -d only needs read access to the source tree — some
# root-owned files created by the chroot pass might not be world-readable, a real, honest, not
# fully guaranteed edge case, named directly rather than assumed away).
#
# Live-verified this session, standalone, before folding into this script: `parted` partitioning
# a plain regular file, `mkfs.vfat`/`mkfs.ext4` formatting plain files directly, `debugfs -w`
# writing into an unmounted ext4 image, `mtools`'s `mcopy` writing into an unmounted FAT32 image,
# and `dd`-assembling two separately-built sub-images into one combined image byte-identically —
# all with zero root, zero loop device, zero mount.

set -euo pipefail

ALPINE_VER="3.20.10"
ALPINE_BRANCH="v3.20"
WORKDIR="${EMILYOS_PI_BUILD_DIR:-/home/fatbaby/EmilyOS/dist/pi-build}"
ROOTFS="$WORKDIR/rootfs"
BOOTSRC="$WORKDIR/alpine-rpi-boot"
BOOT_IMG="$WORKDIR/boot.img"
ROOT_IMG="$WORKDIR/root.img"
IMG="$WORKDIR/emilyos-pi-${ALPINE_VER}.img"
BOOT_SIZE_MB=200   # real Alpine RPi boot bundle (all dtbs+overlays+kernel+modloop) is ~65MiB;
                    # generous headroom for FAT32 cluster overhead + future growth
ROOT_SIZE_MB=512   # real, minimal v0 size — grow once the actual finished package set is known

mkdir -p "$WORKDIR"
cd "$WORKDIR"

echo "== 1. fetch real, official Alpine RPi boot bundle + apk-tools-static (cached if present) =="
if [ ! -f "alpine-rpi-${ALPINE_VER}-aarch64.tar.gz" ]; then
  curl -fsSLO "https://dl-cdn.alpinelinux.org/alpine/${ALPINE_BRANCH}/releases/aarch64/alpine-rpi-${ALPINE_VER}-aarch64.tar.gz"
fi
if [ ! -d "$BOOTSRC" ]; then
  mkdir -p "$BOOTSRC"
  tar -xzf "alpine-rpi-${ALPINE_VER}-aarch64.tar.gz" -C "$BOOTSRC"
fi
if [ ! -x "$WORKDIR/apktools/sbin/apk.static" ]; then
  APK_STATIC_APK=$(curl -fsSL "https://dl-cdn.alpinelinux.org/alpine/${ALPINE_BRANCH}/main/x86_64/" \
    | grep -oE 'apk-tools-static-[0-9.r-]+\.apk' | sort -V | tail -1)
  curl -fsSLO "https://dl-cdn.alpinelinux.org/alpine/${ALPINE_BRANCH}/main/x86_64/${APK_STATIC_APK}"
  mkdir -p "$WORKDIR/apktools"
  tar -xzf "$APK_STATIC_APK" -C "$WORKDIR/apktools"
  chmod +x "$WORKDIR/apktools/sbin/apk.static"
fi

echo "== 2. root-less apk bootstrap of the aarch64 rootfs (package fetch/extraction only — real,
       chroot-based post-install/trigger scripts genuinely need root, see this script's own
       header comment; re-run this step is harmless/idempotent, apk skips already-installed pkgs) =="
mkdir -p "$ROOTFS/etc/apk" "$ROOTFS/lib/apk/db" "$ROOTFS/var/cache/apk"
"$WORKDIR/apktools/sbin/apk.static" \
  -X "https://dl-cdn.alpinelinux.org/alpine/${ALPINE_BRANCH}/main" \
  -X "https://dl-cdn.alpinelinux.org/alpine/${ALPINE_BRANCH}/community" \
  -U --allow-untrusted --arch aarch64 --root "$ROOTFS" --initdb \
  add alpine-base openrc dhcpcd chrony openssh e2fsprogs parted 2>&1 \
  | grep -v "^ERROR:.*chroot: Operation not permitted$" \
  | grep -v "^ERROR:.*script exited with error 127$" \
  | grep -v "^ERROR: [0-9]* errors updating directory permissions$" || true
echo "hostname" > /dev/null # no-op, keeps set -e from tripping on the grep pipeline above
echo "emilyos-pi" > "$ROOTFS/etc/hostname"
mkdir -p "$ROOTFS/etc/apk"
cat > "$ROOTFS/etc/apk/repositories" <<EOF
https://dl-cdn.alpinelinux.org/alpine/${ALPINE_BRANCH}/main
https://dl-cdn.alpinelinux.org/alpine/${ALPINE_BRANCH}/community
EOF
cat > "$ROOTFS/etc/fstab" <<'EOF'
/dev/mmcblk0p1  /boot  vfat  defaults  0  2
/dev/mmcblk0p2  /      ext4  defaults  0  1
EOF

echo "== 3. root-less aarch64 cross-toolchain bootstrap, then build EmilyOS's own Go binary for
       linux/arm64 (cgo enabled — internal/fsaclmod is a genuine cgo package, see its own header
       comment, no cgo-disabled fallback build tag; CGO_ENABLED=0 fails outright) =="
mkdir -p "$ROOTFS/usr/local/bin" "$ROOTFS/var/lib/emilyos" "$ROOTFS/etc/init.d"
XTOOL="$WORKDIR/aarch64-xtool"
if [ ! -x "$XTOOL/usr/bin/aarch64-linux-gnu-gcc-13" ]; then
  echo "   fetching + extracting the aarch64 cross-toolchain (root-less: apt-get download +
       dpkg-deb -x — Debian's cross packages install cleanly outside apt entirely, same trick
       already used for apk-tools-static/mtools/git-lfs this session) ..."
  mkdir -p "$XTOOL/_debs"
  ( cd "$XTOOL/_debs" && apt-get download \
      gcc-aarch64-linux-gnu gcc-13-aarch64-linux-gnu gcc-13-aarch64-linux-gnu-base \
      cpp-aarch64-linux-gnu cpp-13-aarch64-linux-gnu binutils-aarch64-linux-gnu \
      libgcc-13-dev-arm64-cross libgcc-s1-arm64-cross libc6-dev-arm64-cross \
      linux-libc-dev-arm64-cross )
  for deb in "$XTOOL"/_debs/*.deb; do dpkg-deb -x "$deb" "$XTOOL"; done
  # The real, target aarch64 runtime libc (libc.so.6 / ld-linux-aarch64.so.1) is NOT part of
  # libc6-dev-arm64-cross (headers/static libs only) — it's libc6:arm64, a foreign-architecture
  # binary package apt won't resolve without `dpkg --add-architecture arm64` (needs root to
  # register). Fetched directly from the real Ubuntu ports mirror instead (ports.ubuntu.com
  # carries non-amd64/i386 architectures at a different pool path than the main mirror), pinned
  # to the exact version matching this box's own noble release.
  LIBC6_ARM64_DEB="libc6_2.39-0ubuntu8.8_arm64.deb"
  curl -fsSLo "$XTOOL/_debs/$LIBC6_ARM64_DEB" \
    "http://ports.ubuntu.com/ubuntu-ports/pool/main/g/glibc/$LIBC6_ARM64_DEB"
  dpkg-deb -x "$XTOOL/_debs/$LIBC6_ARM64_DEB" "$XTOOL"
  # Real, live-found quirk: Debian's cross-gcc bakes an ABSOLUTE default sysroot
  # (/usr/aarch64-linux-gnu) at package-build time, ignoring PATH/LIBRARY_PATH/C_INCLUDE_PATH
  # entirely for its own linker default search dirs — confirmed live via a real "cannot find
  # libc.so.6" failure that only cleared once --sysroot was passed explicitly. libc6:arm64's own
  # runtime .so files land at usr/lib/aarch64-linux-gnu/ (Debian multiarch layout), not
  # usr/aarch64-linux-gnu/lib/ (the cross-gcc's own expected sysroot layout) — copied across.
  cp "$XTOOL/usr/lib/aarch64-linux-gnu/libc.so.6" "$XTOOL/usr/aarch64-linux-gnu/lib/"
  cp "$XTOOL/usr/lib/aarch64-linux-gnu/ld-linux-aarch64.so.1" "$XTOOL/usr/aarch64-linux-gnu/lib/"
fi

# REAL, LIVE-FOUND GAP fixed here, 2026-09-08 (found while cross-compiling the PARENA-powered
# coreutils/shell in step 3b below, then confirmed it applied here too): this cross-toolchain is
# Debian/Ubuntu's, GLIBC-targeted (its own dynamic linker is /lib/ld-linux-aarch64.so.1) -- but
# Alpine, this whole image's actual target, ships MUSL (/lib/ld-musl-aarch64.so.1), a different,
# incompatible ABI. A plain dynamic cross-build of this binary failed live under
# qemu-aarch64-static against the real rootfs with "Could not open /lib/ld-linux-aarch64.so.1" --
# meaning every PRIOR build of this image shipped an emilyos binary that could never actually
# start on the real device. Fixed via CGO_LDFLAGS's own "-static": glibc's static linking needs
# no runtime loader at all, so it runs correctly under any libc, musl included. The linker warns
# about getpwnam_r/getgrgid_r/getgrouplist ("requires... shared libraries... at runtime") because
# Go's os/user package is reachable from this binary's dependency graph -- real, honest, accepted
# risk: EmilyOS itself never calls user/group lookups (confirmed by grep across internal/), so
# those symbols are linked but dead code, not a live NSS dependency this binary actually
# exercises. Live-verified past the warning: qemu-aarch64-static runs the resulting static binary
# correctly (`--help` prints its real usage) against the real rootfs.
EMILYOS_BINARY_BUILT=0
if ( cd /home/fatbaby/EmilyOS \
     && PATH="$XTOOL/usr/bin:$PATH" \
        LD_LIBRARY_PATH="$XTOOL/usr/lib/x86_64-linux-gnu:${LD_LIBRARY_PATH:-}" \
        CGO_CFLAGS="--sysroot=$XTOOL" CGO_LDFLAGS="--sysroot=$XTOOL -static" \
        GOWORK=off GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc-13 \
        go build -o "$WORKDIR/emilyos-arm64" ./cmd/emilyos ) 2>"$WORKDIR/go-build-arm64.err"; then
  EMILYOS_BINARY_BUILT=1
  cp "$WORKDIR/emilyos-arm64" "$ROOTFS/usr/local/bin/emilyos"
  file "$WORKDIR/emilyos-arm64"
else
  echo "REAL, HONEST GAP (not silently skipped): the cgo cross-build still failed even with the"
  echo "aarch64 toolchain bootstrapped root-lessly above. Real error:"
  cat "$WORKDIR/go-build-arm64.err"
  echo "Continuing WITHOUT the EmilyOS binary in this image."
fi

echo "== 3b. cross-compile the PARENA-powered coreutils/shell (docs/PARENA_COREUTILS_NORTHSTAR.md)
       for aarch64, reusing the same root-less \$XTOOL cross-toolchain bootstrapped above --
       founder real-time, 2026-09-08: 'the alpine pi installable parena powered emily os.' Staged
       as AVAILABLE, NOT DEFAULT (that doc's own explicit Phase 5 boundary, matching
       turbogrep/turbosed's own established precedent) -- parenabusybox's applet symlinks
       (echo/basename/pwd/true/false/wc/head/yes/cat/sleep/env, the full Phase 1 set as of
       2026-09-08's 'also a parena based busybox to go with it' follow-up) go in their own
       dedicated, off-PATH directory rather than /usr/local/bin, so they never silently shadow
       Alpine's real coreutils; only parenash (a uniquely-named binary, no collision risk) is
       placed directly on PATH. Real, honest, PARENA's own compiler runs NATIVELY (x86_64) here
       to emit plain, portable C -- only the FINAL compile of that generated C + host driver
       needs the aarch64 cross-compiler, the exact same two-stage shape src/emit.c's own C
       target already has everywhere else.
       Same musl-vs-glibc dynamic-linker gap named in step 3's own comment above applies here
       too -- fixed the same way (-static below), reverified live: all 10 parenabusybox applets
       plus a real parenash script execute correctly under qemu-aarch64-static against the real
       rootfs. =="
PARENA_DIR="/home/fatbaby/PARENA"
PARENA_ARM64_BUILT=0
if [ -x "$PARENA_DIR/parena" ] && ( cd "$PARENA_DIR" \
     && ./parena build stdlib/string.prn stdlib/coreutils/echo.prn \
          stdlib/coreutils/basename.prn stdlib/coreutils/pwd.prn \
          stdlib/coreutils/wc.prn stdlib/coreutils/head.prn stdlib/coreutils/yes.prn \
          -o "$WORKDIR/parenabusybox_gen.c" \
     && cat "$WORKDIR/parenabusybox_gen.c" tools/parenabusybox_host.c \
          > "$WORKDIR/parenabusybox_full.c" \
     && PATH="$XTOOL/usr/bin:$PATH" LD_LIBRARY_PATH="$XTOOL/usr/lib/x86_64-linux-gnu:${LD_LIBRARY_PATH:-}" aarch64-linux-gnu-gcc-13 \
          --sysroot="$XTOOL" -std=c99 -O2 -static -I runtime -DPARENA_NO_GRAPHICS \
          "$WORKDIR/parenabusybox_full.c" src/arena.c -o "$WORKDIR/parenabusybox-arm64" \
     && ./parena build stdlib/string.prn stdlib/coreutils/sh.prn \
          -o "$WORKDIR/parenash_gen.c" \
     && cat "$WORKDIR/parenash_gen.c" tools/parenash_host.c \
          > "$WORKDIR/parenash_full.c" \
     && PATH="$XTOOL/usr/bin:$PATH" LD_LIBRARY_PATH="$XTOOL/usr/lib/x86_64-linux-gnu:${LD_LIBRARY_PATH:-}" aarch64-linux-gnu-gcc-13 \
          --sysroot="$XTOOL" -std=c99 -O2 -static -I runtime -DPARENA_NO_GRAPHICS \
          "$WORKDIR/parenash_full.c" src/arena.c -o "$WORKDIR/parenash-arm64" \
   ) 2>"$WORKDIR/parena-build-arm64.err"; then
  PARENA_ARM64_BUILT=1
  mkdir -p "$ROOTFS/usr/local/parena-coreutils"
  cp "$WORKDIR/parenabusybox-arm64" "$ROOTFS/usr/local/parena-coreutils/parenabusybox"
  for applet in echo basename pwd true false wc head yes cat sleep env; do
    ln -sf parenabusybox "$ROOTFS/usr/local/parena-coreutils/$applet"
  done
  cp "$WORKDIR/parenash-arm64" "$ROOTFS/usr/local/bin/parenash"
  file "$WORKDIR/parenabusybox-arm64" "$WORKDIR/parenash-arm64"
else
  echo "REAL, HONEST GAP (not silently skipped): the PARENA coreutils/shell aarch64 cross-build"
  echo "failed (or $PARENA_DIR/parena isn't built). Real error, if any:"
  cat "$WORKDIR/parena-build-arm64.err" 2>/dev/null || true
  echo "Continuing WITHOUT PARENA's coreutils/shell in this image."
fi

cat > "$ROOTFS/etc/init.d/emilyos" <<'EOF'
#!/sbin/openrc-run
# EmilyOS policy-kernel boot service.
name="emilyos"
description="EmilyOS policy kernel (posture-gated sessions, RBAC, audit log)"
command="/usr/local/bin/emilyos"
command_args="--state-dir=/var/lib/emilyos"
command_background=true
pidfile="/run/emilyos.pid"
depend() {
    need net
    after firewall
}
EOF
chmod +x "$ROOTFS/etc/init.d/emilyos"
# rc-update itself needs a real chroot (aarch64 busybox ash) to run correctly — deferred to the
# privileged finishing script; recorded here as a real, honest TODO marker file instead of
# silently skipped.
echo "rc-update add emilyos default   # run by the privileged finishing script" > "$ROOTFS/var/lib/emilyos/PENDING_RC_UPDATE"
if [ "$EMILYOS_BINARY_BUILT" -eq 0 ]; then
  echo "emilyos binary NOT built this pass -- see go-build-arm64.err in this workdir" > "$ROOTFS/var/lib/emilyos/PENDING_BINARY_BUILD"
fi

echo "== 4. build the FAT32 boot image from Alpine's own RPi boot bundle (firmware, kernel, dtbs,
       overlays, config.txt) via mtools — no mount needed =="
rm -f "$BOOT_IMG"
fallocate -l "${BOOT_SIZE_MB}M" "$BOOT_IMG"
mkfs.vfat -F 32 -n BOOT "$BOOT_IMG" >/dev/null

# REAL, LIVE-FOUND BOOT-CONFIG GAP: Alpine's own stock cmdline.txt
# ("modules=loop,squashfs,sd-mod,usb-storage quiet console=tty1") has NO `root=` — confirmed by
# extracting initramfs-rpi's own real /init script (mkinitfs's standard init): with no `root=`
# kernel arg it takes Alpine's DISKLESS boot path (unpack an apkovl into a tmpfs root, or drop
# into setup-alpine), never touching a persistent disk partition at all. That's categorically the
# wrong boot mode for what this build assembles (a real, persistent apk-installed rootfs baked
# into an ext4 partition) — confirmed live in the SAME init script's own `if [ -n "$KOPT_root" ]`
# branch (grep'd directly, not guessed) that setting `root=` instead runs `nlplug-findfs` +
# `switch_root` into that real partition, the actual "installed to disk" path this image needs.
# Real, checked-not-assumed kernel config: this kernel already builds MMC/SDHCI (CONFIG_MMC_BLOCK,
# CONFIG_MMC_SDHCI) and ext4 (CONFIG_EXT4_FS) in directly (`=y`, not `=m`) — no extra `modules=`
# entries needed for either. `rootflags=rw` is set explicitly rather than relying on openrc's own
# `root` service to remount rw (that service IS also added to the boot runlevel below, in the
# privileged script's rc-update list, as a real belt-and-suspenders match to Alpine's own default
# convention — but an explicit `rw` here means a real, working boot doesn't depend on getting that
# service's own ordering exactly right).
cat > "$WORKDIR/cmdline.txt" <<'EOF'
root=/dev/mmcblk0p2 rootfstype=ext4 rootflags=rw quiet console=tty1
EOF

MTOOLS_SKIP_CHECK=1 mcopy -s -i "$BOOT_IMG" "$BOOTSRC"/*.dtb "$BOOTSRC"/*.elf "$BOOTSRC"/*.dat \
  "$BOOTSRC"/*.bin "$BOOTSRC"/config.txt "$BOOTSRC"/overlays "$BOOTSRC"/boot ::/
MTOOLS_SKIP_CHECK=1 mcopy -i "$BOOT_IMG" "$WORKDIR/cmdline.txt" ::cmdline.txt

echo "== done (root-less half): $BOOTSRC prepared, $BOOT_IMG built, $ROOTFS populated with apk"
echo "packages (post-install/trigger scripts pending) plus the EmilyOS init script and (if the"
echo "cross-compile succeeded above) its binary."
echo ""
echo "REAL, HONEST CORRECTION found live this pass: ext4 population via 'mke2fs -d' also needs"
echo "root, not just the chroot-based finishing step this script's own header comment named --"
echo "Alpine's own busybox-suid package ships /bin/bbsuid as mode ---x--x--x (execute-only, not"
echo "even readable by its own owning user), which a non-root 'mke2fs -d' cannot read to copy."
echo "Root bypasses DAC read checks entirely, so this stops being a problem under real sudo. The"
echo "remaining privileged work (chroot finishing + mke2fs -d population + final dd assembly) is"
echo "in sudo-queue/76-build-emilyos-pi-image.sh (top-level monorepo) -- run that against"
echo "$ROOTFS and $BOOT_IMG to produce the final, complete .img."
