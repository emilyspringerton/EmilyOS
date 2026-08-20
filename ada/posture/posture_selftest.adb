--  Standalone Ravenscar-profile selftest for the Posture module.
--
--  Not a general-purpose test framework -- a small, self-contained main
--  procedure exercising the real transition/capability tables against
--  known-correct answers transcribed directly from internal/posture/
--  machine.go, the same way this whole module was written. Runs as one
--  continuous walk through real states (Guarded_State is a single
--  protected object, not a type -- there is no "reset" to test states
--  out of order), checking capability verdicts while actually in each
--  relevant state rather than after leaving it.
--
--  UNVERIFIED: no GNAT toolchain existed in this monorepo before this
--  session (see posture.ads's own doc comment); this file has not been
--  compiled or run. gnat.adc in this directory carries the Ravenscar
--  profile pragma for the whole partition when this is built via
--  posture.gpr.

with Ada.Text_IO; use Ada.Text_IO;
with Posture;     use Posture;
with Posture.Machine;

procedure Posture_Selftest is

   Failures : Natural := 0;

   procedure Check (Condition : in Boolean; Message : in String) is
   begin
      if Condition then
         Put_Line ("PASS: " & Message);
      else
         Put_Line ("FAIL: " & Message);
         Failures := Failures + 1;
      end if;
   end Check;

   Old : Posture_State;
   Ok  : Boolean;

begin
   Put_Line ("=== Posture module selftest ===");

   Check (Posture.Machine.Current = Normal,
          "starts in Normal, matching machine.go's New() default");
   Check (Posture.Machine.Verdict (Cap_Net) = Pass_Through,
          "Normal posture: every capability reads Pass_Through "
          & "(the real overrides table has an empty Normal row)");

   --  Normal -> Siege is allowed; check Siege's real overrides while
   --  actually in Siege.
   Posture.Machine.Transition (Siege, Old, Ok);
   Check (Ok and then Old = Normal,
          "Normal -> Siege is allowed and reports the real old state");
   Check (Posture.Machine.Current = Siege,
          "current state actually became Siege");
   Check (Posture.Machine.Verdict (Cap_Net) = Force_Off,
          "Siege posture: Cap_Net is Force_Off, matching the real table");
   Check (Posture.Machine.Verdict (Cap_Domain_Start) = Pinned_Only,
          "Siege posture: Cap_Domain_Start is Pinned_Only");
   Check (Posture.Machine.Verdict (Cap_Exec) = Pass_Through,
          "Siege posture: an unlisted capability (Cap_Exec) still "
          & "correctly falls back to Pass_Through, not an error");

   --  Siege -> Mercy is NOT in the real transition table.
   Posture.Machine.Transition (Mercy, Old, Ok);
   Check (not Ok,
          "Siege -> Mercy is correctly rejected (not in the real table)");
   Check (Posture.Machine.Current = Siege,
          "a rejected transition leaves the state unchanged");

   --  Siege -> Normal is allowed; return to a known state.
   Posture.Machine.Transition (Normal, Old, Ok);
   Check (Ok, "Siege -> Normal is allowed");

   --  GAME toggle: Normal -> Game -> Game must land on Normal, not stay
   --  in Game or error.
   Posture.Machine.Transition (Game, Old, Ok);
   Check (Ok and then Posture.Machine.Current = Game,
          "Normal -> Game is allowed");
   Check (Posture.Machine.Verdict (Cap_Domain_Start) = Game_Domain_Only,
          "Game posture: Cap_Domain_Start is Game_Domain_Only, matching "
          & "the real table");
   Posture.Machine.Transition (Game, Old, Ok);
   Check (Ok and then Posture.Machine.Current = Normal,
          "Game -> Game (the toggle) actually lands on Normal, "
          & "matching machine.go's explicit special case");

   --  Incident: check its real overrides (including the one Force_On,
   --  Cap_Export) while actually in Incident, before it becomes
   --  unreachable behind Exited.
   Posture.Machine.Transition (Incident, Old, Ok);
   Check (Ok, "Normal -> Incident is allowed");
   Check (Posture.Machine.Verdict (Cap_Net) = Force_Off,
          "Incident posture: Cap_Net is Force_Off");
   Check (Posture.Machine.Verdict (Cap_Export) = Force_On,
          "Incident posture: Cap_Export is Force_On -- the one real "
          & "ForceOn override in the whole table (incident evidence "
          & "export always allowed, matching machine.go's own inline "
          & "comment)");

   --  Exited is terminal: nothing transitions out of it, and its own
   --  capability row is intentionally all Pass_Through (the real
   --  lockout is enforced by the verb dispatcher checking
   --  Current = Exited outright, a different code path -- see
   --  posture-machine.adb's own comment on this).
   Posture.Machine.Transition (Exited, Old, Ok);
   Check (Ok, "Incident -> Exited is allowed");
   Posture.Machine.Transition (Normal, Old, Ok);
   Check (not Ok and then Posture.Machine.Current = Exited,
          "nothing transitions out of Exited, ever -- confirmed for "
          & "the one real permitted-looking case (back to Normal) too");
   Check (Posture.Machine.Verdict (Cap_Net) = Pass_Through,
          "Exited posture: Cap_Net reads Pass_Through here, matching "
          & "machine.go's own comment that Exited's real lockout is "
          & "enforced elsewhere, not via this table");

   Put_Line ("");
   if Failures = 0 then
      Put_Line ("ALL PASS");
   else
      Put_Line (Natural'Image (Failures) & " FAILURE(S)");
   end if;
end Posture_Selftest;
