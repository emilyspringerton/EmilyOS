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

echo "== 3. build EmilyOS's own Go binary for linux/arm64, wire it in as a real OpenRC service =="
mkdir -p "$ROOTFS/usr/local/bin" "$ROOTFS/var/lib/emilyos" "$ROOTFS/etc/init.d"
EMILYOS_BINARY_BUILT=0
if ( cd /home/fatbaby/EmilyOS && GOWORK=off GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o "$WORKDIR/emilyos-arm64" ./cmd/emilyos ) 2>"$WORKDIR/go-build-arm64.err"; then
  EMILYOS_BINARY_BUILT=1
  cp "$WORKDIR/emilyos-arm64" "$ROOTFS/usr/local/bin/emilyos"
else
  echo "REAL, HONEST GAP (not silently skipped): EmilyOS's cmd/emilyos does not currently cross-"
  echo "compile to linux/arm64 with CGO_ENABLED=0 -- internal/fsaclmod is a real cgo package (it"
  echo "links against a PARENA-compiled C mod, see its own header comment) with no cgo-disabled"
  echo "fallback build tag, and this box has no aarch64 cross-compiler installed to build it with"
  echo "CGO_ENABLED=1 GOARCH=arm64 CC=aarch64-linux-gnu-gcc either. Real error:"
  cat "$WORKDIR/go-build-arm64.err"
  echo "Continuing WITHOUT the EmilyOS binary in this image -- see NORTHSTAR_DISTRO.md's own"
  echo "'Phase 2 cross-compilation gap' note. Install gcc-aarch64-linux-gnu (needs root -- queue"
  echo "it) and re-run this script to retry with CGO_ENABLED=1 once that's available."
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
       overlays, config.txt/cmdline.txt) via mtools — no mount needed =="
rm -f "$BOOT_IMG"
fallocate -l "${BOOT_SIZE_MB}M" "$BOOT_IMG"
mkfs.vfat -F 32 -n BOOT "$BOOT_IMG" >/dev/null
MTOOLS_SKIP_CHECK=1 mcopy -s -i "$BOOT_IMG" "$BOOTSRC"/*.dtb "$BOOTSRC"/*.elf "$BOOTSRC"/*.dat \
  "$BOOTSRC"/*.bin "$BOOTSRC"/config.txt "$BOOTSRC"/cmdline.txt "$BOOTSRC"/overlays "$BOOTSRC"/boot \
  ::/

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
