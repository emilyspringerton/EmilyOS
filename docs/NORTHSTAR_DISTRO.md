# EmilyOS — Installable Distro North Star (scoping only, no implementation)

**Status:** Draft v0.1 — scoping only, no implementation.
**Date:** 2026-08-25.

## Why this exists

Founder, real-time, in a rapid burst while EmilyOS's existing `GRANT-FS`/`REVOKE-FS` filesystem-
ACL work (S191-07, S193-02) was already dispatched: *"oh yea cool lets build our own installable
distro"* → *"EmilyOS for real as arch linux"* → *"PARENA native as much as possible"* → *"we use
gnu tools for the load bearing walls like grep"* → *"but the stuff we dont need leave it out"* →
*"make it super small to start whatever can be left out leave it out"* → *"build parena in"* →
*"and vim"* → *"and emily cli."*

This is a real, substantial reversal of EmilyOS's own existing `NORTHSTAR.md`, which states as a
deliberate architectural choice: *"It is not a bare-metal OS — it is the policy kernel that runs
on Linux."* Per Emily Way Principle 2 (spec before implementation), this document exists to
capture the pivot honestly and flag its real open questions **before** anything gets built —
this was explicitly not handed to a fork blind, unlike the narrower GRANT-FS work already in
flight, because "build an installable Linux distribution" is categorically bigger (bootloader,
partitioning, ISO remastering, package selection) than any single mod-surface feature this
session has built so far.

## The actual shape, reconciled with the existing north star

This is not "EmilyOS's own Go policy-kernel binary becomes bare-metal." EmilyOS's existing stack
(Go 1.22+, systemd for domain lifecycle, Ubuntu/Debian baseline) stays exactly what it is. What's
new: **a real, installable Arch-based Linux distribution that ships with EmilyOS's policy kernel
baked in as the OS's own security/audit layer** — the RBAC, capability gates, hash-chained audit
log, and posture state machine EmilyOS already built for a *hosted* service become the actual
operating environment's security model, not just a service running on top of a generic distro.
That's a real evolution of "the policy kernel that runs on Linux" into "the policy kernel the OS
is built around," not a contradiction of the existing SOC 2-readiness mission — arguably a more
complete expression of it (an auditor attesting to an entire purpose-built OS's access control is
a stronger claim than attesting to one service's).

## Concrete design guidance already given, real not guessed

1. **GNU tools stay for load-bearing infrastructure.** "we use gnu tools for the load bearing
   walls like grep" — matches this monorepo's own Principle 17 (Load-Bearing) framing directly:
   coreutils, grep, bash, systemd, the kernel itself — proven, correctness-critical pieces don't
   get experimentally swapped for PARENA-native alternatives just because PARENA-native is the
   session's wider theme. `turbosed`/`turbogrep` (this session's own work) are the concrete
   precedent for the right relationship: PARENA tooling as an *available, opt-in, fallback-safe*
   layer alongside real GNU tools, not a wholesale replacement of them — turbogrep still isn't
   symlinked over real `grep` by default even after real verification work; turbosed explicitly
   isn't either, for the same reason. This distro should ship both, same relationship.
2. **Minimal by default.** "but the stuff we dont need leave it out" / "make it super small to
   start whatever can be left out leave it out" — a genuinely minimal base image, not a
   full-featured desktop spin. This is a natural fit for Arch specifically: Arch's own real
   design philosophy (KISS — Keep It Simple, minimal defaults, the user builds up from a bare
   base) is already aligned with this ask, not something to fight against or layer on top of.
3. **Named included-by-default set, real and specific**: PARENA ("build parena in"), vim ("and
   vim"), the `emily` CLI ("and emily cli"), SSH ("and ssh" — matches PITVIPER's own established
   "i am always in ssh" philosophy, the default persistent mode this whole ecosystem already
   assumes, not a new idea), systemd ("systemd" — already EmilyOS's own stated stack for domain
   lifecycle management, and Arch's own real default init system anyway, not an addition so much
   as a confirmation), and the compiler toolchain both of those actually need — GCC ("build in gcc") and GNAT/Ada ("build in ada and gnat"). The Ada requirement isn't
   arbitrary: `EmilyOS/ada/posture/` already has a real, existing Ada implementation of the
   posture state machine (`posture-machine.adb`/`.ads`, `posture_selftest.adb`) — a deliberate
   language choice for a SOC 2/policy-kernel context (Ada's own real track record in
   high-assurance/safety-critical systems), and GNAT (GCC's Ada frontend, already installable via
   `sudo-queue/17-install-gnat.sh` on this dev box) is what compiles it. GCC itself is also what
   PARENA's own C-emission target ultimately compiles down to. Everything else is deliberately
   out unless it's load-bearing GNU infra (point 1) or the base Arch install genuinely requires it
   to boot/network/authenticate.

## Real open questions — not decided here, need a founder call or a real scoping pass

1. **Base mechanism**: a from-scratch ISO build (real `archiso`/`mkarch` tooling), or a
   configuration/package-list layer on top of a stock Arch install (closer to how many
   Arch-derivatives — EndeavourOS, Artix — actually work: not a fully independent distro, an
   opinionated installer + package set on real upstream Arch)? The latter is dramatically less
   engineering work and inherits Arch's own security-update cadence directly; the former gives
   more control over exactly what ships but is a much bigger, more failure-prone undertaking
   (bootloader config, partitioning logic, driver/firmware bundling all become this project's own
   responsibility instead of upstream Arch's).
2. **Target hardware**: is this for the same future ThinkPad already mentioned this session
   (alongside the MAC-spoofing tooling), or for this VPS/future VPS instances too, or both? A
   laptop install (real hardware: WiFi drivers, battery/power management, a real display) and a
   headless VPS install (no GPU/display concerns, but real cloud-provider network/firmware
   quirks) are genuinely different scoping problems — not decided which (or both) this is for.
3. **EmilyOS-as-init-layer, concretely**: does the policy kernel run as a normal systemd service
   on top of a standard Arch boot (simpler, closer to what already exists), or does it hook
   earlier in the boot sequence (PID 1 adjacent, an actual "policy kernel" in the more literal
   sense EmilyOS's own name implies)? The existing `NORTHSTAR.md` milestones (audit foundation,
   RBAC, posture) were all built assuming a service running on a generic Linux host — whether
   they need real changes to work as more deeply "the OS itself" isn't established here.
4. **"PARENA native as much as possible" — where, concretely?** System scripts and utilities
   this distro itself needs (setup/config tooling, not user-facing apps) are the most natural
   first target, following the same "mod-surface API first" MO already established for GFD/
   PITVIPER/REDGARDEN this session — but "as much as possible" is aspirational language, not a
   scoped list. A real next pass should name specific pieces (the installer's own scripting? boot
   hooks? the package-selection tool itself?) rather than treating "PARENA native" as a blanket
   goal applied to everything at once.
5. **Relationship to `stdlib/container/*` and `stdlib/pentest/*`'s own existing design work** —
   `STDLIB.md` already has real, if implementation-blocked, design passes for `container/lxc`/
   `container/cgroup` and the pentest toolkit; a real installable distro is a natural consumer of
   both (container tooling for isolation, pentest tooling as an included package matching this
   session's own `macspoof.prn` work) — not integrated or cross-referenced here, flagged as a
   real connection point for whoever picks this up next.

## Pivot: Alpine, targeting Raspberry Pi (2026-09-08)

Founder, real-time: *"ok can we start working on an installable alpine based raspi distro I
guess in Emily os repo."* This resolves two of the four open questions above, not a rejection of
the rest of this document — the GNU-load-bearing-walls / minimal-by-default / PARENA-native /
included-package guidance from 2026-08-25 all still applies, base distro aside. Confirmed with
the founder directly before writing anything: **EmilyOS becomes the real distro** (not a generic
distro that happens to run EmilyOS's binary, and not just image-build plumbing with the branding
question deferred) — this repo's own name stops being aspirational, and `image-builder-rpi`
(S213)'s already-forked tooling is the real thing to adapt, with `FLASH` (S213) staying the
separate "write the finished image to an SD card" tool it already is.

**Open question 1 (base mechanism), resolved by a real technical constraint, not preference**:
Arch has no first-class official ARM/Raspberry Pi story — "Arch Linux ARM" is a separate,
community-maintained project, not upstream Arch itself, which is what made question 1
(`archiso`/`mkarch` vs. an installer-on-real-Arch) genuinely hard to answer for a Pi target.
Alpine, by contrast, ships a real, official, first-class `aarch64` release train with a
Pi-specific flavor — checked live, not assumed (`https://dl-cdn.alpinelinux.org/alpine/v3.20/
releases/aarch64/alpine-rpi-3.20.10-aarch64.tar.gz`, downloaded and inspected in full this pass).
That artifact is a complete real boot-partition bundle: RPi firmware (`bootcode.bin`, `start.elf`/
`start4.elf`, `fixup.dat`/`fixup4.dat`), a prebuilt `vmlinuz-rpi` kernel + `initramfs-rpi` +
`modloop-rpi` (kernel modules as a squashfs), and real device-tree blobs/overlays covering the
actual current Pi hardware matrix: Zero 2 (W), 3B/3B+/3A+/CM3, 4B/400/CM4, 5B/CM5 — plus a
`config.txt`/`cmdline.txt` pair and a real `apks/aarch64/*.apk` package cache. **Real, decisive
finding: this makes "from-scratch ISO/kernel build" a non-question for the Pi target** — Alpine
upstream already builds and ships the real, tested kernel+firmware+dtb layer; there is no reason
to rebuild any of that ourselves, only to assemble a root filesystem and an image around it. This
directly extends the `PARENA-0001` finding already logged elsewhere in this monorepo's own
BACKLOG (PARENA's C emitter + runtime already compile and link clean, fully static, under a musl
toolchain — Alpine's real libc) — Alpine was already the musl-portability target, now it's also
the real Pi target.

**Open question 2 (target hardware), resolved**: Raspberry Pi, not the ThinkPad/VPS branches
that question originally named — those stay open, unaffected, for whenever they're picked up.

**Real, live root-less build-path proof (this pass, no root, no Docker, no VM)**: Alpine
publishes `apk-tools-static` for `x86_64` (the same real bootstrap mechanism Alpine's own
official Docker base-image build uses to cross-bootstrap an `aarch64` rootfs from a non-Alpine,
non-ARM host). Downloaded it, extracted `apk.static`, and ran a real
`--root <dir> --arch aarch64 --initdb add alpine-base openrc` against the real upstream `v3.20`
`main`/`community` repos. Result: **24/24 packages fetched and extracted correctly** (~17 MiB,
real `/etc/os-release` reporting `Alpine Linux v3.20.10`) — the package-fetch/extraction half of
rootfs assembly needs no privilege at all. **Real, honest boundary found live**: every package's
`chroot`-based post-install/trigger script (busybox, alpine-baselayout, openrc) failed with
`chroot: Operation not permitted`, and a bulk "104 errors updating directory permissions" pass
also failed — both need real `CAP_SYS_CHROOT`/ownership privilege this sandbox's own `uid=1000`
doesn't have. Checked two real, standard privilege-less workarounds before concluding this is a
genuine, unavoidable boundary here: `fakeroot` is installed but only fakes `stat`/`chown`-family
syscalls, not `chroot(2)` itself, so it doesn't help the failing post-install scripts;
`unshare --user --map-root-user` (unprivileged Linux user namespaces, which the kernel itself
has enabled — confirmed via `/proc/sys/kernel/unprivileged_userns_clone` = `1`) is additionally
blocked by this specific sandbox's own container policy (`write failed /proc/self/uid_map:
Operation not permitted`) — a real, separate restriction beyond plain root-vs-non-root. Neither
`proot` nor `bubblewrap` is installed here either. **Conclusion, named honestly rather than
worked around with something fragile**: the rootfs-finishing step and the actual image assembly
(FAT32 boot partition + ext4 root partition into one flashable `.img`) need a real privileged
pass — queued the same way every other root-requiring step in this monorepo already is
(`sudo-queue/`), not attempted with a workaround likely to silently produce a broken image.

**Real, root-less pieces already usable for the privileged script when it runs**: `debugfs -w`
(present on this box) can write files into an ext4 image directly without mounting it — a real,
already-established technique for building filesystem images without root; the FAT32 boot
partition needs `mtools` (`mcopy`/`mformat`), not installed here, installable either via
`sudo-queue` or via the same root-less `apt-get download` + `dpkg-deb -x` extraction trick
`PARENA-0001`'s own musl-toolchain work already used successfully this monorepo.

## Refinement: shrinking the privileged surface, live-tested (2026-09-08, same-day follow-up)

Went further before just queuing the whole build behind root: proved live that `parted` can
partition a **plain regular file** directly (no `losetup`, no loop device — confirmed via
`parted -s file.img mklabel msdos` + `mkpart`, `print` showing exact byte offsets), that
`mkfs.vfat`/`mkfs.ext4` format plain files the same way, that `mtools`'s `mcopy` writes into an
unmounted FAT32 image file with zero mount, and that `dd`-copying two independently-built
sub-images into one combined file at exact byte offsets is byte-identical to building them in
place (checked via `cmp` against both partitions after assembly). This meant the ENTIRE image
layout/assembly step — not just package fetch — could move out of the privileged half.

Acted on it: split the single monolithic privileged script into a new root-less script,
`EmilyOS/packaging/scripts/build-pi-image-rootless.sh`, and ran it for real (not just written —
executed end to end in this sandbox): fetches the same `alpine-rpi-3.20.10-aarch64.tar.gz` +
`apk-tools-static`, does the same root-less `apk` package bootstrap already proven above, builds
the real FAT32 `boot.img` from Alpine's own boot bundle via `mtools` (verified live: `mdir`
listing shows real `bcm2710-rpi-3-b.dtb`/`bcm2712-rpi-5-b.dtb`/etc. inside it), and attempts
EmilyOS's own Go cross-build. Two further real, honest gaps found live while actually running it,
neither previously named, both folded back into the (now smaller) remaining privileged script
(`sudo-queue/76-build-emilyos-pi-image.sh`, top-level monorepo) rather than worked around:

1. **`mke2fs -d <dir>` also needs root**, not just the chroot-based post-install step — found
   live via a real "Permission denied while opening 'bbsuid'" failure: Alpine's own
   `busybox-suid` package ships `/bin/bbsuid` as mode `---x--x--x` (execute-only, unreadable even
   by its own owning user) — a real, intentional hardening convention on Alpine's part, not a
   bug. A non-root reader genuinely cannot copy that file's contents to build the ext4 image;
   root bypasses DAC read checks entirely, so this is a non-issue once genuinely privileged.
2. **EmilyOS's own `cmd/emilyos` does not cross-compile to `linux/arm64` with `CGO_ENABLED=0`** —
   found live: `internal/fsaclmod` (EmilyOS's PARENA-mod-backed GRANT_FS/REVOKE_FS package,
   2026-08-25) is a genuine `cgo` package with no cgo-disabled fallback build tag, so disabling
   cgo for a static cross-build excludes all its Go files entirely (`build constraints exclude
   all Go files`). **Resolved root-lessly the same day**, closing what was first queued as a
   privileged step: Debian's `gcc-aarch64-linux-gnu` cross-toolchain packages (and their real
   dependency chain — `gcc-13-aarch64-linux-gnu`, `binutils-aarch64-linux-gnu`,
   `libgcc-13-dev-arm64-cross`, `libgcc-s1-arm64-cross`, `libc6-dev-arm64-cross`) install cleanly
   via the same root-less `apt-get download` + `dpkg-deb -x` extraction already used for
   `apk-tools-static`/`mtools`/`git-lfs` this session — cross-compilers are just files, `apt`
   itself is the only part that needs root, and only for writing into `/var/lib/dpkg`, not for
   anything the compiler itself does. Two real, live-found wrinkles, neither guessable in
   advance: (a) the actual target-architecture runtime libc (`libc.so.6`, `ld-linux-aarch64.so.1`)
   is NOT part of `libc6-dev-arm64-cross` (headers/static libs only) — it's `libc6:arm64`, a
   foreign-architecture binary package `apt` won't resolve without `dpkg --add-architecture arm64`
   (needs root to register), fetched instead directly from the real Ubuntu ports mirror
   (`ports.ubuntu.com` carries non-amd64/i386 architectures at a different pool path than the
   main mirror), version-pinned to exactly match this box's own noble release; (b) Debian's
   cross-gcc bakes an ABSOLUTE default sysroot (`/usr/aarch64-linux-gnu`) at package-build time,
   ignoring `PATH`/`LIBRARY_PATH`/`C_INCLUDE_PATH` entirely for the linker's own default search
   dirs — confirmed live via a real "cannot find libc.so.6" failure that only cleared once
   `--sysroot=<extraction dir>` was passed explicitly via `CGO_CFLAGS`/`CGO_LDFLAGS`. With both
   fixed, `GOWORK=off GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc-13 go build`
   produces a real, correct `ELF 64-bit LSB executable, ARM aarch64, ... dynamically linked,
   interpreter /lib/ld-linux-aarch64.so.1` — live-verified via `file`, not just a clean exit code.
   Folded into `build-pi-image-rootless.sh` itself (a one-time toolchain bootstrap, cached for
   re-runs); the privileged script no longer touches the Go build at all.

Net result: the privileged script no longer does ANY loop-mounting, `losetup`, filesystem
mounting, or Go cross-compilation at all — its real, remaining scope is exactly the two things
checked live as genuinely requiring root (chroot-based package finishing + `qemu-user-static`
registration, and `mke2fs -d` against the finished tree), plus the same
root-less `parted`+`dd` assembly technique reused rather than re-invented under sudo.

## Real, live-found boot-config gap fixed before it could waste a privileged run (2026-09-08, third same-day pass)

Caught before ever running the privileged script, by reading the real `initramfs-rpi` init
script directly (extracted and grepped, not assumed): Alpine's own stock `cmdline.txt`
(`modules=loop,squashfs,sd-mod,usb-storage quiet console=tty1`) has **no `root=` at all** — with
none set, mkinitfs's real `init` takes Alpine's DISKLESS boot path (unpack an apkovl into a
tmpfs root), never touching a persistent disk partition. That is categorically the wrong boot
mode for an apk-installed rootfs baked into a real ext4 partition, which is exactly what this
build assembles. Confirmed live in the same init script's own `if [ -n "$KOPT_root" ]` branch:
setting `root=` instead runs `nlplug-findfs` + `switch_root` into that real partition — the
actual disk-install path this image needs. Fixed: `build-pi-image-rootless.sh` now writes its
own `cmdline.txt` (`root=/dev/mmcblk0p2 rootfstype=ext4 rootflags=rw quiet console=tty1`) instead
of copying Alpine's stock diskless one verbatim. Checked the real kernel config before adding any
`modules=` entries: MMC/SDHCI and ext4 are both built directly into this kernel (`CONFIG_MMC_
BLOCK=y`, `CONFIG_MMC_SDHCI=y`, `CONFIG_EXT4_FS=y`, not `=m`), so none are needed.

Same pass, found by cross-checking the built rootfs's own real `/etc/init.d/` listing against
this script's existing `rc-update` calls: `root`, `fsck`, `localmount`, `swap`, `seedrng` were
missing entirely — real, standard Alpine boot-runlevel services a disk-installed system needs
(`root` remounts the kernel-mounted root per fstab and runs pending fsck; `fsck`/`localmount`
handle everything else in `/etc/fstab`, here just `/boot`). Added to `sudo-queue/76`'s own
`rc-update` list (real names confirmed present in the rootfs, not guessed) as a belt-and-
suspenders match to Alpine's own documented default, alongside the explicit `rootflags=rw` above
(so a working boot doesn't depend on getting `root`'s own service ordering exactly right).

Neither of these is boot-tested yet — this sandbox has no real Pi hardware or
`qemu-system-aarch64` (Phase 3, below, still not started) — but both are real, live-checked
fixes against the actual init script and the actual rootfs contents, not guesses, and catching
them here means the first real privileged run has a much better chance of producing something
that actually boots, not just something that assembles cleanly.

## Phase 3 attempted, real partial result — no visible boot console yet (2026-09-08, fourth same-day pass)

Bootstrapped `qemu-system-aarch64` root-lessly the same way as everything else this session
(`apt-get download` + `dpkg-deb -x`, real deps: `qemu-system-arm`/`-common`/`-data` plus
`libbrlapi0.8`/`libcacard0`/`libfdt1`/`libpmem1`/`librdmacm1t64`/`libslirp0`/`liburing2`/
`libusbredirparser1t64`/`libndctl6`/`libdaxctl1`, the last two found only by iterating real
"cannot open shared object file" errors, not predicted up front). Confirmed a real, working
binary (`qemu-system-aarch64 --version` runs clean) with real `raspi0`/`raspi1ap`/`raspi2b`/
`raspi3ap`/`raspi3b` machine models available (this QEMU version — 8.2.2 — has no `raspi4b`).

Booted the real `vmlinuz-rpi` + `initramfs-rpi` + `bcm2710-rpi-3-b.dtb` (all three already
verified present and correct from the root-less build) under `-M raspi3b` with the corrected
`root=`-bearing cmdline. **Real, partial, honest result**: no visible console output reached
stdio across several real variants tried (`-serial stdio`, `-nographic`, `-serial mon:stdio`,
with and without an explicit `earlycon=pl011,mmio32,0x3f215040` kernel argument) — but QEMU's own
`-d guest_errors,unimp` debug log confirms the kernel IS actively executing and probing real
hardware (35 real log lines: `bcm2835_property` mailbox tag requests, `bcm2836_control_write`
register accesses, `dwc2_glbreg_write` USB controller reset attempts) — a genuine, live signal
that the kernel+dtb pairing boots far enough to start real hardware initialization, not an
immediate crash or hang at instruction zero. The most likely real explanation for the missing
console text, not yet confirmed: Alpine's stock kernel may default its actual boot-time console
to the framebuffer (matching the ORIGINAL stock `cmdline.txt`'s own `console=tty1`, a screen
target, not a UART) rather than either RPi UART (`ttyAMA0`/PL011 typically routed to Bluetooth on
real Pi 3 hardware, `ttyS0`/mini-UART routed to the GPIO header) — QEMU's raspi3b framebuffer
emulation, if incomplete or requiring `-display` output this pass ran without (`-display none`
throughout, to keep the test scriptable/headless), would produce exactly this "kernel clearly
running, zero visible text" symptom. Real, honest, not-yet-closed follow-up, not silently
dropped: confirm which console the kernel actually targets (check `/proc/cmdline`-equivalent
defaults or `dmesg` once real console output IS achieved some way), and/or try a VNC/framebuffer
display target instead of `-display none` to rule the framebuffer theory in or out.
**Phase 3 stays open** — this pass improved confidence (real hardware-level execution confirmed)
without yet producing the actual pass/fail boot signal Phase 3 exists to get.

## Phase 3, continued same day — the framebuffer theory confirmed, and a real, definitive boot-pipeline pass

Confirmed the framebuffer theory directly rather than leaving it as a guess: QEMU's own QMP
`screendump` command (`{"execute":"screendump","arguments":{"filename":"..."}}` over the
`-qmp unix:...` socket, `nc -U` to speak the protocol) captures the emulated device's actual
framebuffer content regardless of `-display none` — no VNC client, no GUI, needed. First
screendump (no disk attached, plain `console=tty1`) showed real, live kernel boot text
(`raspberrypi-exp-gpio ... Failed to get GPIO N config`, `bcm2835_vchiq ... failed to set
channelbase`, etc. — all real, known QEMU raspi3b hardware-emulation gaps for peripherals this
build doesn't need, not fatal) followed by `Mounting boot media failed. initramfs emergency
recovery shell launched.` — the exact correct behavior for a run with no boot media attached.

Built a real (if throwaway) root-less test image to go one step further: the SAME real `boot.img`
this session's own root-less script already produces, `dd`-assembled (same proven technique)
alongside a tiny SYNTHETIC ext4 "root" partition (one file, `/etc/os-release`, no real init) —
purely to test the partition table + `cmdline.txt` + mount + `switch_root` pipeline structurally,
not to test a real bootable system. Attached via `-drive file=...,if=sd` (QEMU's own SD-card
emulation requires a power-of-2 image size — `truncate -s 256M`, a real, live-found constraint).

**Real, definitive result, screendump saved at `docs/pi-boot-test-switch-root-panic.png`**:

```
switch_root: can't execute '/sbin/init': No such file or directory
[   11.705544] Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000100
[...]
Hardware name: Raspberry Pi 3 Model B (DT)
```

This is exactly the correct, expected failure for a synthetic root with no real `/sbin/init` —
and it proves, live, that every real piece of this build's own boot pipeline works correctly end
to end: the kernel finds and mounts the real `/dev/mmcblk0p2` ext4 partition (confirming
`cmdline.txt`'s own `root=/dev/mmcblk0p2 rootfstype=ext4 rootflags=rw` is correct), `nlplug-
findfs` resolves it, and `switch_root` genuinely transitions into it (confirmed by the kernel
correctly trying and failing to exec `/sbin/init` FROM that new root, not the initramfs's own).
The panic itself is the kernel's own standard, correct response to PID 1 exiting — not a bug.

**Real, honest conclusion**: the actual Alpine rootfs this session's own root-less+privileged
scripts build (once the privileged half actually runs) DOES ship a real `/sbin/init`
(`busybox`/`openrc`'s own, installed via `alpine-base`/`openrc` — already confirmed present in
the built `rootfs/` tree, not assumed) at exactly the path this test just proved `switch_root`
correctly looks for. This doesn't yet prove the REAL rootfs boots all the way to a working
login/SSH/`emilyos` service — that still needs the actual privileged run and a real boot test
against its real output — but it closes out the one class of risk that would have been hardest
to debug AFTER a failed real attempt (a wrong `cmdline.txt`/partition layout silently producing a
non-booting image with no clear error). That risk is now retired with real, positive evidence,
not just careful reasoning.

**Real phased plan from here**:
- Phase 0 (this pass, done): placement decision, Alpine-for-Pi technical justification, real
  boot-artifact inventory, root-less rootfs-bootstrap proof and its real privilege boundary.
- Phase 1: a real, single `sudo-queue/` script doing the full privileged build — `apk` install
  into a scratch root (finishing the chroot-based post-installs this pass couldn't), assembling
  the real FAT32 boot + ext4 root image from Alpine's own `alpine-rpi` release bundle plus that
  rootfs, producing one genuinely flashable `.img`.
- Phase 2: wire EmilyOS's own Go binary into the image as a real OpenRC service (`/etc/init.d/
  emilyos`) — resolves this document's own open question 3 (service-on-boot vs. PID-1-adjacent)
  in the simpler direction for v0, matching the founder's own "make it super small to start"
  guidance; a deeper boot-sequence hook stays a real, later option, not required for a first
  bootable image.
- Phase 3: boot-test the image — needs either real Pi hardware or `qemu-system-aarch64` (not
  installed in this sandbox, named honestly, not assumed available).
- Phase 4: hand the finished `.img` to `FLASH` (S213) for the real "write to an SD card" step,
  closing the loop with the existing thread instead of duplicating its scope.
- Phase 5+: the 2026-08-25 package-selection guidance (PARENA, vim, `emily` CLI, SSH, GCC,
  GNAT/Ada for `EmilyOS/ada/posture/`) layered onto the working image, plus the
  `stdlib/container/*`/`stdlib/pentest/*` connective tissue this document already named.

## Phase 3, continued again same day — the REAL rootfs boots into OpenRC, root-lessly confirmed

Went one step further than the synthetic-root test above: does the REAL, apk-installed rootfs
(minus the chroot-based finishing pass this sandbox genuinely can't do) boot any further than
`switch_root: can't execute '/sbin/init'`? Copied the real, built `rootfs/` tree (excluding the
one real, execute-only `/bin/bbsuid` file `cp` itself can't read either — same real DAC
limitation named earlier, unrelated to this test) and booted it as-is: **identical failure**,
`can't execute '/sbin/init': No such file or directory` — the real rootfs genuinely has no
`/sbin/init` yet, confirming live (not just theorized) that the privileged chroot-finishing step
is load-bearing, not optional.

Found the exact real reason by reading busybox's own actual `.trigger` script (extracted directly
from the real `busybox-1.36.1-r31.apk`, not guessed): its real, load-bearing action is
`/bin/busybox --install -s`, which creates busybox's own ~304 applet symlinks (`init`, `sh`,
`mount`, `md5sum`, `getty`, etc.) — the exact script that fails root-lessly (`chroot: Operation
not permitted`) in this session's own earlier bootstrap. This is REAL, direct confirmation of
what the privileged step actually does and why it's necessary — not just an assumption.

Went further still, root-lessly: root-lessly bootstrapped `qemu-user-static`'s
`qemu-aarch64-static` (same `apt-get download` + `dpkg-deb -x` technique) and used it for a
genuinely SAFE, READ-ONLY check — `qemu-aarch64-static -L <rootfs> <rootfs>/bin/busybox --list`
— to confirm live that the real rootfs's own `busybox` binary really does have an `init` applet
(304 real applet names printed, `init` and `sh` both confirmed present), WITHOUT ever executing
`--install` itself (which does ABSOLUTE-PATH filesystem writes — running it without a real
chroot would write into THIS HOST's own real `/sbin`, `/bin`, etc., a genuinely unacceptable
side effect this pass deliberately avoided, not attempted).

Instead, on a THROWAWAY test copy only (never the real, committed `rootfs/`), created just the
one real symlink busybox's own installer would (`sbin/init -> ../bin/busybox`, matching Alpine's
own relative-symlink convention) by hand via plain `ln -s` — no binary execution, no filesystem-
root ambiguity, completely safe. **Real, definitive, screendump-captured result** (saved at
`docs/pi-boot-test-openrc-starting.png`):

```
OpenRC 0.54 is starting up Linux 6.6.49-0-rpi (aarch64)
* md5sum is missing, which suggests /usr is not mounted
* /lib/rc/sh/init.sh: line 76: mount: not found
* Caching service dependencies ...
* Clock skew detected with `/etc/init.d/emilyos`
* can't run '/sbin/getty'
```

**Real, honest interpretation**: `switch_root` now succeeds, busybox's `init` applet correctly
detects the real OpenRC installation and launches its real startup sequence, and OpenRC caches
service dependencies AND explicitly finds `/etc/init.d/emilyos` — the real EmilyOS service file
this session's own scripts wired in. The remaining errors (`md5sum`/`mount`/`getty` "not found")
are a direct, expected ARTIFACT of this test's own deliberate shortcut — only ONE of busybox's
~304 real applet symlinks was created by hand, not the full set `busybox --install -s` creates
for real. They are not evidence of a bug in the real build; they're exactly what's expected to
disappear once the actual privileged `apk fix` (which re-runs busybox's real trigger script,
creating all 304 real symlinks) runs for real. This test doesn't replace that real privileged
run — it forecasts, with real, live evidence rather than hope, that the actual build should get
meaningfully further than "won't boot at all," probably to a working service/login state.

## Status

Phase 0 done. Phase 1 (root-less half) done and re-verified clean from scratch. Phase 3 (boot-
test) has real, live, positive evidence at three levels now: the boot pipeline (cmdline/mount/
switch_root) works, `init`/OpenRC start correctly once `/sbin/init` exists, and OpenRC finds the
real `emilyos` service — all confirmed via QEMU screendumps, not assumed. The one remaining real
gate before a first genuinely complete, bootable `.img` exists is still the actual privileged
run — `sudo-queue/76-build-emilyos-pi-image.sh` (top-level monorepo), which does the real,
full `busybox --install -s` (via `apk fix`) this session's own test could only partially,
carefully hand-simulate. Queue it via `sudo-queue/`, matching this monorepo's own established
convention for any step needing root this sandbox doesn't have.
