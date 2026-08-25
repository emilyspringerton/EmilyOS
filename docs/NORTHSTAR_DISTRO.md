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
   vim"), the `emily` CLI ("and emily cli"). Everything else is deliberately out unless it's
   load-bearing GNU infra (point 1) or the base Arch install genuinely requires it to boot/network/
   authenticate.

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

## Status

Scoping only. No milestone plan, no ISO, no installer script. Real next step, per the founder's
own established "spec before implementation" discipline: resolve question 1 (from-scratch ISO vs.
installer-on-real-Arch) before any implementation work starts — it's the single decision every
other piece of scope depends on.
