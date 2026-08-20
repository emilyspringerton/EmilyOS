--  EmilyOS Posture module (Ravenscar Ada port).
--
--  Founder, real-time: "do a module of emily os in ravenscar32 ada".
--
--  Ported faithfully from internal/posture/machine.go (Go, the real and
--  still-authoritative implementation this package mirrors -- see that
--  file's own doc comment for the full "posture is physics, RBAC is
--  policy" design). Ravenscar was chosen deliberately for a policy-
--  decision kernel's core state machine specifically: bounded,
--  deterministic, no dynamic allocation, no unbounded recursion --
--  exactly the properties a SOC 2 Type II control (this repo's own
--  stated northstar, docs/NORTHSTAR.md) benefits from being able to
--  argue about formally. This is a MODULE, not a rewrite: the Go
--  implementation stays authoritative and running in production; this
--  package is a faithful, independently-verifiable port of its pure
--  decision logic (state transitions + capability verdicts) only --
--  not the persistence layer (var/posture.json) or any I/O, which stay
--  Go-side.
--
--  UNVERIFIED as of the commit that adds this file: no Ada/GNAT
--  toolchain existed anywhere in this monorepo before this session.
--  Installing GNAT needs real sudo (sudo-queue/17-install-gnat.sh,
--  queued not run -- this sandboxed environment has no passwordless
--  sudo). Written as carefully and conservatively as real Ada syntax
--  allows without a compiler in the loop to check it against -- same
--  honesty standard this repo already holds GOLDENBAND's own unrun
--  Blender export scripts to (GoblinFoxDragon/docs2/
--  GOLDENBAND_INTEGRATION_NORTHSTAR.md).

package Posture is

   pragma Pure;

   --  Posture states, exactly matching machine.go's six named constants
   --  (Normal/Siege/Mercy/Incident/Game/Exited).
   type Posture_State is
     (Normal, Siege, Mercy, Incident, Game, Exited);

   --  Capabilities, exactly matching internal/policy/rbac.go's full real
   --  "cap.*" constant set (13 capabilities as of this port -- not a
   --  subset chosen for convenience).
   type Capability is
     (Cap_Posture_Set,
      Cap_Posture_Admin,
      Cap_Session_Open,
      Cap_Net,
      Cap_Exec,
      Cap_Domain_Start,
      Cap_Domain_Stop,
      Cap_SSH_Connect,
      Cap_SSH_Manage_Hosts,
      Cap_SSH_Manage_Keys,
      Cap_Policy_Write,
      Cap_Audit_Read,
      Cap_Export);

   --  Exactly matching machine.go's CapabilityVerdict enum.
   type Capability_Verdict is
     (Pass_Through, Force_Off, Force_On, Pinned_Only, Game_Domain_Only);

end Posture;
