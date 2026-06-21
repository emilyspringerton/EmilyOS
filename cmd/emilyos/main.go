// emilyos — EmilyOS policy kernel CLI
//
// Usage:
//
//	emilyos posture get
//	emilyos posture set <state>          (requires EMILY_ROLE=admin)
//	emilyos verb dispatch <verb> <object>
//	emilyos audit tail [-n N]
//	emilyos audit verify
//
// Environment:
//
//	EMILY_ACTOR_ID     — identity performing the action (default: $USER)
//	EMILY_SESSION_ID   — session identifier (default: pid-based)
//	EMILY_DEVICE_ID    — device identifier (default: hostname)
//	EMILY_ROLE         — operator | admin | auditor (default: operator)
//	EMILY_POSTURE_PATH — path to posture.json (default: var/posture.json)
//	EMILY_AUDIT_PATH   — path to audit log (default: var/audit.jsonl)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"emilyos/internal/audit"
	"emilyos/internal/policy"
	"emilyos/internal/posture"
	"emilyos/internal/verb"
)

var version = "dev"

func main() {
	flag.Usage = usage
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("emilyos %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 2 {
		usage()
		os.Exit(1)
	}

	switch args[0] {
	case "posture":
		runPosture(args[1:])
	case "verb":
		runVerb(args[1:])
	case "audit":
		runAudit(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `emilyos - EmilyOS policy kernel

Commands:
  posture get                    print current posture state
  posture set <state>            transition posture (admin only)
  verb dispatch <verb> <object>  dispatch a verb
  audit tail [-n N]              print last N audit events (default 10)
  audit verify                   verify audit chain integrity

Flags:
  --version   print version and exit

Environment:
  EMILY_ACTOR_ID     identity (default: $USER)
  EMILY_SESSION_ID   session id (default: pid)
  EMILY_DEVICE_ID    device (default: hostname)
  EMILY_ROLE         operator|admin|auditor (default: operator)
  EMILY_POSTURE_PATH path to posture.json (default: var/posture.json)
  EMILY_AUDIT_PATH   path to audit.jsonl (default: var/audit.jsonl)
`)
}

func runPosture(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: emilyos posture get|set <state>")
		os.Exit(1)
	}
	pm, err := posture.New(posturePath())
	if err != nil {
		die("posture: %v", err)
	}
	switch args[0] {
	case "get":
		fmt.Println(pm.Current())
	case "set":
		if len(args) < 2 {
			die("posture set: state required")
		}
		target := args[1]
		ctx := callerContext()
		if ctx.Role != policy.RoleAdmin {
			die("posture set requires admin role (current: %s)", ctx.Role)
		}
		log, err := audit.Open(auditPath())
		if err != nil {
			die("audit: %v", err)
		}
		defer log.Close()
		old, err := pm.Transition(target, ctx.ActorID, ctx.SessionID)
		if err != nil {
			_ = log.Deny(ctx.ActorID, ctx.SessionID, ctx.DeviceID,
				"posture.set", "posture:"+target, "transition.invalid:"+err.Error(), nil)
			die("transition failed: %v", err)
		}
		_ = log.Allow(ctx.ActorID, ctx.SessionID, ctx.DeviceID,
			"posture.set", "posture:"+target, audit.ResultSuccess,
			map[string]any{"from": old, "to": target})
		fmt.Printf("%s → %s\n", old, target)
	default:
		die("unknown posture subcommand: %s", args[0])
	}
}

func runVerb(args []string) {
	if len(args) < 1 || args[0] != "dispatch" {
		fmt.Fprintln(os.Stderr, "usage: emilyos verb dispatch <verb> <object>")
		os.Exit(1)
	}
	if len(args) < 3 {
		die("verb dispatch: <verb> and <object> required")
	}
	verbName := args[1]
	objectRef := args[2]

	log, err := audit.Open(auditPath())
	if err != nil {
		die("audit: %v", err)
	}
	defer log.Close()

	pm, err := posture.New(posturePath())
	if err != nil {
		die("posture: %v", err)
	}

	d := verb.New(log, pm)
	d.Register(verbName, capForVerb(verbName), func(ctx verb.Context, objectRef string, meta map[string]any) error {
		fmt.Printf("verb %s dispatched on %s\n", verbName, objectRef)
		return nil
	})

	ctx := callerContext()
	if err := d.Dispatch(ctx, verbName, objectRef, nil); err != nil {
		if verb.IsDenied(err) {
			fmt.Fprintf(os.Stderr, "denied: %v\n", err)
			os.Exit(2)
		}
		die("dispatch: %v", err)
	}
}

// capForVerb maps common verb names to required capabilities.
func capForVerb(v string) string {
	if envCap := os.Getenv("EMILY_VERB_CAP"); envCap != "" {
		return envCap
	}
	switch v {
	case "ENTER", "RESUME", "PAUSE", "EXIT", "WITHDRAW", "GAME":
		return policy.CapSessionOpen
	case "EXEC", "DOMAIN_EXEC":
		return policy.CapExec
	case "NET", "SSH":
		return policy.CapNet
	case "INCIDENT":
		return policy.CapPostureAdmin
	case "EXPORT":
		return policy.CapExport
	case "POLICY_CHANGE":
		return policy.CapPolicyWrite
	case "AUDIT_READ":
		return policy.CapAuditRead
	case "DOMAIN_START":
		return policy.CapDomainStart
	case "DOMAIN_STOP":
		return policy.CapDomainStop
	case "SSH_MANAGE_HOSTS":
		return policy.CapSSHManageHosts
	case "SSH_MANAGE_KEYS":
		return policy.CapSSHManageKeys
	default:
		return policy.CapExec
	}
}

func runAudit(args []string) {
	if len(args) == 0 {
		die("usage: emilyos audit tail|verify")
	}
	switch args[0] {
	case "tail":
		n := 10
		fs := flag.NewFlagSet("audit tail", flag.ExitOnError)
		fs.IntVar(&n, "n", 10, "number of events to show")
		_ = fs.Parse(args[1:])
		if fs.NArg() > 0 {
			if v, err := strconv.Atoi(fs.Arg(0)); err == nil {
				n = v
			}
		}
		events, err := audit.ReadFile(auditPath())
		if err != nil {
			die("read audit: %v", err)
		}
		start := len(events) - n
		if start < 0 {
			start = 0
		}
		for _, e := range events[start:] {
			b, _ := json.Marshal(e)
			fmt.Println(string(b))
		}
	case "verify":
		if err := audit.VerifyChain(auditPath()); err != nil {
			fmt.Fprintf(os.Stderr, "TAMPERED: %v\n", err)
			os.Exit(3)
		}
		events, _ := audit.ReadFile(auditPath())
		fmt.Printf("ok — %d events, chain intact\n", len(events))
	default:
		die("unknown audit subcommand: %s", args[0])
	}
}

// callerContext builds a verb.Context from env vars.
func callerContext() verb.Context {
	actorID := getenvOr("EMILY_ACTOR_ID", os.Getenv("USER"))
	if actorID == "" {
		actorID = "unknown"
	}
	sessionID := getenvOr("EMILY_SESSION_ID", fmt.Sprintf("pid-%d", os.Getpid()))
	deviceID, _ := os.Hostname()
	if envDevice := os.Getenv("EMILY_DEVICE_ID"); envDevice != "" {
		deviceID = envDevice
	}
	role := getenvOr("EMILY_ROLE", policy.RoleOperator)
	if !policy.ValidRole(role) {
		fmt.Fprintf(os.Stderr, "warning: unknown role %q, defaulting to operator\n", role)
		role = policy.RoleOperator
	}
	return verb.Context{
		ActorID:   actorID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		Role:      role,
	}
}

func posturePath() string { return getenvOr("EMILY_POSTURE_PATH", "var/posture.json") }
func auditPath() string   { return getenvOr("EMILY_AUDIT_PATH", "var/audit.jsonl") }

func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "emilyos: "+format+"\n", args...)
	os.Exit(1)
}
