--  Posture.Machine -- the protected state machine itself.
--
--  A Ravenscar-profile protected object replaces machine.go's
--  sync.RWMutex-guarded struct: Ravenscar restricts general task
--  rendezvous (no Select/Abort/dynamic task creation) but fully
--  supports and encourages protected objects for exactly this shape of
--  problem -- one piece of shared mutable state, a handful of bounded
--  mutually-exclusive operations on it, no unbounded blocking. See
--  posture.ads for the full module-scope rationale.

package Posture.Machine is

   --  Attempts to move to State_To. On success, Old_State is the state
   --  the machine was in immediately before the call and Success is
   --  True. On a rejected transition (either the (From, To) pair isn't
   --  permitted, or the machine has already reached Exited, which is
   --  terminal), Old_State is still the pre-call state, and Success is
   --  False -- mirrors machine.go's Transition returning (oldState,
   --  err) rather than raising, so a caller always gets a real answer
   --  without needing exception handling for the expected/common
   --  "transition not allowed" case.
   --
   --  GAME is a toggle, exactly matching machine.go: calling
   --  Transition (To => Game) while already in Game moves to Normal
   --  instead, handled internally before the transition table is
   --  consulted.
   procedure Transition
     (State_To  : in  Posture_State;
      Old_State : out Posture_State;
      Success   : out Boolean);

   --  The current posture. Pure query, no side effect.
   function Current return Posture_State;

   --  How the current posture treats Cap. Pure query -- matches
   --  machine.go's CapabilityVerdict method exactly, including that
   --  Exited has no capability overrides recorded here (every
   --  capability reads Pass_Through in the Exited state); machine.go's
   --  own comment notes Exited's total capability lockout is enforced
   --  by a different code path entirely (the verb dispatcher checking
   --  Current = Exited outright), not through this table -- preserved
   --  faithfully here rather than inventing a second enforcement path
   --  this module wouldn't actually match production behavior on.
   function Verdict (Cap : Capability) return Capability_Verdict;

private

   protected Guarded_State is
      procedure Set (State_To  : in  Posture_State;
                      Old_State : out Posture_State;
                      Success   : out Boolean);
      function Get return Posture_State;
   private
      Current_State : Posture_State := Normal;
   end Guarded_State;

end Posture.Machine;
