EmilyOS 2 GUI v0.1 — Game-UI Filesystem
Core premise

The game interface is the filesystem UI.

Interaction is intent-declared (no single-click actions).

Visual language is hard-coded palette + hard-coded affordance types.

1) Visual system
1.1 Background + whitespace rules

Background: deep navy/near-black (no pure black required, but close).

Directories: the only use of “white-ish” is EGSHELL (directory tiles / directory labels / directory frames).

No other whites anywhere else (no white buttons, no white highlights).

1.2 Tile types

Two primary tile classes:

A) Directory tiles (EGSHELL / white-ish)

Must read as “container / safe / navigational.”

Any frame/border for directories also uses EGSHELL.

B) Button tiles (colored)

Must be darker than EGSHELL always.

Colors allowed: blue, magenta, teal, light teal, dark teal, red, yellow, green
(You also said “all HTML colors like olive excluding white”—so the rule is: allow a standard HTML color set, but ban white & near-white; enforce luminance cap.)

1.3 Typography

Labels mimic the WINNER text aesthetic from the mockup:

blocky, geometric, high legibility

all-caps by default (or small caps look)

Directory labels: EGSHELL or slightly dimmer EGSHELL.

Button labels: high-contrast vs button fill, but never pure white.

1.4 Visual feedback

Hover effects are allowed only as focus outlines, not “clickable hover glow.”

Focus outline color:

For directories: EGSHELL outline

For buttons: a slightly brighter tint of the button’s own color

No animations that imply “aliveness.” If anything animates, it must be:

single-shot (e.g., a brief flash on activation)

no looping pulses

2) Layout model: tmux × i3 hybrid
2.1 Screen is a tiling space

The UI is composed of panes (rectangles) like tmux/i3.

Panes can contain:

a directory view (grid of tiles)

a file view (grid of tiles)

a “process/log” panel (optional)

a “command line / verb bar” panel (optional)

2.2 Pane operations (keyboard-first)

Create/split panes:

vertical split

horizontal split

Move focus between panes

Resize pane boundaries

Close pane (non-destructive; doesn’t delete data)

2.3 No freeform dragging by default

Dragging is optional, but if included:

it must be a deliberate verb (“MOVE MODE”) not casual click-drag.

3) Interaction contract
3.1 No single-click affordances

Single click never triggers actions.
Single click only does:

focus tile

select tile (highlight)

prime for double click

3.2 Double click semantics (two speeds)

You defined two double-click flavors:

Fast double click = ACTIVATE

activates the tile’s primary action (button press / open directory / open file)

uses the tile’s “associated action”

Slow double click = EDIT LABEL

enters label editing mode (inline rename)

applies to both files and directories (unless tile is “system-locked”)

3.3 Timing thresholds (implementable)

Pick deterministic timing (example values—tune later, but lock shape now):

DC_FAST_MAX = 220ms between clicks

DC_SLOW_MIN = 350ms and DC_SLOW_MAX = 800ms

Anything between 221–349ms: treat as fast (or require a “dead zone” if you want strictness)

Important: fast and slow are mutually exclusive and must not misfire.

3.4 Editing mode behavior

When in label-edit mode:

keystrokes edit the label

ENTER commits

ESC cancels

clicking outside does not commit; it cancels (safer / less accidental)

No background autosave while typing; commit is explicit.

4) Filesystem mapping to tiles
4.1 Tile representations

Every file/dir is a tile with:

type (dir/file/button/system)

label

color (from palette rules)

action (primary)

meta (size, time, hash) optionally hidden behind an info verb

4.2 Directories

Directory tiles (EGSHELL) open into a directory pane view.

Directory panes display:

child tiles (dirs first, then files)

stable ordering (no jitter)

optional filter/search is allowed but must be a verb, not auto

4.3 Files as buttons

“File tiles” are button-like colored squares (unless you explicitly want files visually distinct from buttons; if you do, add a small corner glyph)

Fast double click:

if file is executable → run (inside its domain/posture rules)

if file is data → open viewer pane (read-only by default)

Slow double click:

rename file (unless locked)

5) Safety + policy hooks (EmilyOS posture compatibility)
5.1 Postures affect available actions, not the visuals

Under SIEGE/MERCY:

tiles remain visible

activation may be denied with a minimal feedback (see below)

5.2 Denial feedback (non-alive, non-chatty)

When an action is denied:

one-frame “deny flash” on the tile border (e.g., dim red outline)

no modal dialogs

no toast notifications

optional: a quiet status line update in a dedicated pane (single line)

6) Keyboard navigation (must exist)

Even if mouse exists, keyboard is first-class:

Arrow keys / WASD moves selection within the grid

TAB cycles panes

ENTER is equivalent to fast double click on the focused tile

F2 (or R) enters rename (equivalent to slow double click)

ESC backs out / cancels / returns to stillness depending on context

7) What’s intentionally missing (v0 scope control)

No single-click actions

No drag-drop unless explicitly invoked as a verb

No animated UI chrome

No freeform window stacking (tiling only)

No popups/modals

If you want the next step, I can convert this into a concrete event/state spec (mouse events → interpreter → verb dispatch), including an unambiguous “double-click classifier” pseudocode so it behaves exactly the same across Windows/Linux/SDL.
