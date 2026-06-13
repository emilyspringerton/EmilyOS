SPEC 5
 
emilyspringerton edited this page on Jan 31 · 1 revision
Acknowledged. Writing the skeleton.
This will be minimal, authoritative, and bootable. No scaffolding. No future hooks. Just the irreversibility threshold.---

Done.
This is the irreversibility threshold. Four files:

boot.c — 400 lines, zero scaffolding
UEFI entry → framebuffer → exit boot services → main loop
3×5 line font embedded as read-only kernel data
Immediate-mode line drawing (Bresenham)
Agency verb FSM (defined, not wired)
HLT loop (presence without response)
Makefile — Clean build, no extras
Compiles to emily.efi
Links against GNU-EFI
No debug symbols, no bloat
README.md — Build, test, philosophy check
QEMU test instructions
Real hardware install path
Current state vs. intentional gaps
Philosophy enforcement checklist
test.sh — One command to boot
Builds, creates disk image, launches QEMU
You'll see: black screen, white "emily", silence
What This Proves
If this boots cleanly:

You can take over the machine
Framebuffer control works
Font rendering is legible
System is silent (HLT works)
Foundation is mechanically sound
If it doesn't feel calm, stop. The rest won't fix it.

What's Not Here (and Why)
Missing	Reason
Keyboard input	Stub only—proves concept first
Interrupt setup	Not needed until input is real
Miner	Comes after state machine works
Page tables	Designed but not enforced yet
ACPI shutdown	Halts instead (simpler for testing)
This is intentional minimalism. Each piece comes after the previous one feels right.

The skeleton is written. Build it. Boot it. If the screen shows only "emily" and nothing moves, you've crossed the threshold.
