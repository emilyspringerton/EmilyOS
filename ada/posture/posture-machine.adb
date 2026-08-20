package body Posture.Machine is

   --  Transition permission table, transcribed directly from machine.go's
   --  `transitions` map. True at (From, To) means that move is allowed.
   --  Ada's array-over-enum type makes every (From, To) pair explicit --
   --  no pair can be silently missing the way a sparse Go map entry
   --  could be, which is exactly the "make the implicit default
   --  explicit" property this whole module exists to get from the port.
   --
   --  Exited has no outgoing transitions at all (it's terminal) -- in
   --  machine.go this is enforced by an explicit early check
   --  ("if m.current == Exited { return error }") rather than the table
   --  itself, but recording it as an all-False row here too means this
   --  table is correct on its own terms, not only correct because of an
   --  early-return elsewhere -- a real property Go's version doesn't
   --  have, since its own `transitions` map has no Exited row at all
   --  and would return an implicit false either way. Preserved as a
   --  deliberate strengthening of the port, not a behavior change: both
   --  implementations still refuse every transition out of Exited.
   type Transition_Table is
     array (Posture_State, Posture_State) of Boolean;

   Allowed : constant Transition_Table :=
     (Normal   => (Siege | Mercy | Game | Incident | Exited => True,
                   Normal                                    => False),
      Siege    => (Normal | Game | Exited => True,
                   others                 => False),
      Mercy    => (Normal | Siege | Game | Exited => True,
                   others                          => False),
      Incident => (Normal | Exited => True,
                   others           => False),
      Game     => (Normal | Exited => True,
                   others           => False),
      Exited   => (others => False));

   --  Capability override table, transcribed directly from machine.go's
   --  `postureCapOverrides` map. A (State, Cap) pair not explicitly
   --  listed there reads Pass_Through -- represented here as every cell
   --  defaulting to Pass_Through via `others`, then only the real,
   --  named overrides from the Go source set to anything else.
   type Verdict_Table is
     array (Posture_State, Capability) of Capability_Verdict;

   Overrides : constant Verdict_Table :=
     (Normal   => (others => Pass_Through),
      Siege    => (Cap_Net          => Force_Off,
                    Cap_Domain_Start => Pinned_Only,
                    others           => Pass_Through),
      Mercy    => (Cap_Exec         => Pinned_Only,
                    Cap_Domain_Start => Pinned_Only,
                    others           => Pass_Through),
      Incident => (Cap_Net          => Force_Off,
                    Cap_Exec         => Force_Off,
                    Cap_Domain_Start => Force_Off,
                    Cap_Export       => Force_On,
                    others           => Pass_Through),
      Game     => (Cap_Net          => Force_Off,
                    Cap_Domain_Start => Game_Domain_Only,
                    Cap_Exec         => Game_Domain_Only,
                    others           => Pass_Through),
      Exited   => (others => Pass_Through));
      --  Exited's row is all Pass_Through here, exactly matching
      --  machine.go's own comment on its Exited map entry: "no
      --  capabilities allowed in exited state (handled separately)" --
      --  that lockout is the verb dispatcher's job (checking
      --  Current = Exited outright), not this table's, in both
      --  implementations.

   ------------------
   -- Guarded_State --
   ------------------

   protected body Guarded_State is

      procedure Set (State_To  : in  Posture_State;
                      Old_State : out Posture_State;
                      Success   : out Boolean)
      is
         Target : Posture_State := State_To;
      begin
         Old_State := Current_State;

         if Current_State = Exited then
            Success := False;
            return;
         end if;

         --  GAME is a toggle: GAME while already in GAME returns to
         --  NORMAL, exactly matching machine.go's
         --  "if m.current == Game && to == Game { to = Normal }".
         if Current_State = Game and then State_To = Game then
            Target := Normal;
         end if;

         if Allowed (Current_State, Target) then
            Current_State := Target;
            Success := True;
         else
            Success := False;
         end if;
      end Set;

      function Get return Posture_State is
        (Current_State);

   end Guarded_State;

   ----------------
   -- Transition --
   ----------------

   procedure Transition
     (State_To  : in  Posture_State;
      Old_State : out Posture_State;
      Success   : out Boolean)
   is
   begin
      Guarded_State.Set (State_To, Old_State, Success);
   end Transition;

   -------------
   -- Current --
   -------------

   function Current return Posture_State is
     (Guarded_State.Get);

   -------------
   -- Verdict --
   -------------

   function Verdict (Cap : Capability) return Capability_Verdict is
     (Overrides (Guarded_State.Get, Cap));

end Posture.Machine;
