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
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"emilyos/internal/audit"
	"emilyos/internal/policy"
	"emilyos/internal/posture"
	"emilyos/internal/verb"
)

var version = "dev"

// Build attestation — injected at build time via -ldflags:
//   -X main.buildCommit=$(git rev-parse --short HEAD)
//   -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)
var buildCommit = "unknown"
var buildDate = "unknown"

func main() {
	flag.Usage = usage
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("emilyos %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
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
	case "about":
		runAbout()
	case "snapshot":
		runSnapshot(args[1:])
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
	case "export":
		if len(args) < 2 {
			die("audit export: <outdir> required")
		}
		runAuditExport(args[1])
	case "bundle":
		outPath := "soc2-evidence.tar.gz"
		if len(args) >= 2 {
			outPath = args[1]
		}
		runAuditBundle(outPath)
	case "history":
		n := 20
		if len(args) >= 2 {
			if v, err := strconv.Atoi(args[1]); err == nil && v > 0 {
				n = v
			}
		}
		runAuditHistory(n)
	default:
		die("unknown audit subcommand: %s", args[0])
	}
}

// runAuditBundle produces a tar.gz SOC 2 evidence bundle containing:
//   - audit.jsonl (full audit log)
//   - policy-snapshot.json (latest policy snapshot, if any)
//   - manifest.json (file list with SHA-256 hashes, chain_ok, build info)
func runAuditBundle(outPath string) {
	// Verify chain first.
	events, _ := audit.ReadFile(auditPath())
	chainOK := true
	if verifyErr := audit.VerifyEvents(events); verifyErr != nil {
		fmt.Fprintf(os.Stderr, "warning: chain verification failed: %v\n", verifyErr)
		chainOK = false
	}

	// Collect files.
	type bundleFile struct {
		name string
		data []byte
	}
	var files []bundleFile

	// audit.jsonl
	auditData, _ := os.ReadFile(auditPath())
	files = append(files, bundleFile{"audit.jsonl", auditData})

	// latest policy snapshot
	ss, _ := policy.NewSnapshotStore(snapshotPath())
	if latestSnap, _ := ss.Latest(); latestSnap != nil {
		snapData, _ := json.MarshalIndent(latestSnap, "", "  ")
		files = append(files, bundleFile{"policy-snapshot.json", append(snapData, '\n')})
	}

	// Compute SHA-256 for each file.
	type fileEntry struct {
		Name   string `json:"name"`
		Bytes  int    `json:"bytes"`
		SHA256 string `json:"sha256"`
	}
	var entries []fileEntry
	for _, f := range files {
		sum := sha256.Sum256(f.data)
		entries = append(entries, fileEntry{
			Name:   f.name,
			Bytes:  len(f.data),
			SHA256: hex.EncodeToString(sum[:]),
		})
	}

	// Build manifest.
	manifest := map[string]any{
		"export_ts":    time.Now().UTC().Format(time.RFC3339),
		"event_count":  len(events),
		"chain_ok":     chainOK,
		"build_commit": buildCommit,
		"build_id":     version + "+" + buildDate,
		"actor":        callerContext().ActorID,
		"files":        entries,
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	files = append(files, bundleFile{"manifest.json", append(manifestData, '\n')})

	// Write tar.gz.
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		die("create bundle: %v", err)
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	tw := tar.NewWriter(gw)
	for _, f := range files {
		hdr := &tar.Header{
			Name:    f.name,
			Size:    int64(len(f.data)),
			Mode:    0o640,
			ModTime: time.Now().UTC(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			die("tar header: %v", err)
		}
		if _, err := tw.Write(f.data); err != nil {
			die("tar write: %v", err)
		}
	}
	_ = tw.Close()
	_ = gw.Close()

	fmt.Printf("bundle written: %s\n", outPath)
	fmt.Printf("  events: %d  chain_ok: %v\n", len(events), chainOK)
	for _, e := range entries {
		fmt.Printf("  %-30s  %s  (%d bytes)\n", e.Name, e.SHA256[:16]+"…", e.Bytes)
	}
	if !chainOK {
		os.Exit(3)
	}
}

// runAuditExport writes audit.jsonl + manifest.json to outdir for SOC 2 evidence.
func runAuditExport(outdir string) {
	if err := os.MkdirAll(outdir, 0o750); err != nil {
		die("create outdir: %v", err)
	}

	// Read and verify chain.
	src := auditPath()
	events, err := audit.ReadFile(src)
	if err != nil {
		die("read audit: %v", err)
	}
	chainOK := true
	if verifyErr := audit.VerifyEvents(events); verifyErr != nil {
		fmt.Fprintf(os.Stderr, "warning: chain verification failed: %v\n", verifyErr)
		chainOK = false
	}

	// Copy audit.jsonl to outdir.
	destLog := filepath.Join(outdir, "audit.jsonl")
	if err := copyFile(src, destLog); err != nil {
		die("copy audit log: %v", err)
	}

	// Determine first/last seq.
	var firstSeq, lastSeq int64
	if len(events) > 0 {
		firstSeq = events[0].Seq
		lastSeq = events[len(events)-1].Seq
	}

	// Write manifest.
	manifest := map[string]any{
		"export_ts":   time.Now().UTC().Format(time.RFC3339),
		"event_count": len(events),
		"first_seq":   firstSeq,
		"last_seq":    lastSeq,
		"chain_ok":    chainOK,
		"source":      src,
		"actor":       callerContext().ActorID,
	}
	manifestPath := filepath.Join(outdir, "manifest.json")
	mdata, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, append(mdata, '\n'), 0o640); err != nil {
		die("write manifest: %v", err)
	}

	fmt.Printf("exported %d events → %s\n", len(events), outdir)
	fmt.Printf("  %s\n", destLog)
	fmt.Printf("  %s  (chain_ok=%v)\n", manifestPath, chainOK)
	if !chainOK {
		os.Exit(3)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			// Empty log is OK — create empty file.
			return os.WriteFile(dst, nil, 0o640)
		}
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// runAuditHistory prints the last N posture.set events from the audit log.
func runAuditHistory(n int) {
	events, err := audit.ReadFile(auditPath())
	if err != nil {
		die("read audit: %v", err)
	}

	type row struct {
		Seq    int64
		TS     string
		Actor  string
		From   string
		To     string
	}

	var rows []row
	for _, e := range events {
		if e.Verb != "posture.set" {
			continue
		}
		from, _ := e.Meta["from"].(string)
		to, _ := e.Meta["to"].(string)
		rows = append(rows, row{
			Seq:   e.Seq,
			TS:    e.TS.Format("2006-01-02 15:04:05"),
			Actor: e.ActorID,
			From:  from,
			To:    to,
		})
	}

	if len(rows) == 0 {
		fmt.Println("no posture transitions recorded")
		return
	}
	if len(rows) > n {
		rows = rows[len(rows)-n:]
	}

	fmt.Printf("%-6s  %-19s  %-16s  %s\n", "Seq", "Timestamp", "Actor", "Transition")
	fmt.Println(strings.Repeat("─", 65))
	for _, r := range rows {
		fmt.Printf("%-6d  %-19s  %-16s  %s → %s\n", r.Seq, r.TS, r.Actor, r.From, r.To)
	}
}

// runAbout prints build attestation and current policy state.
func runAbout() {
	fmt.Printf("emilyos %s\n", version)
	fmt.Printf("  commit:  %s\n", buildCommit)
	fmt.Printf("  built:   %s\n", buildDate)
	pm, _ := posture.New(posturePath())
	if pm != nil {
		fmt.Printf("  posture: %s\n", pm.Current())
	}

	ss, _ := policy.NewSnapshotStore(snapshotPath())
	if latest, _ := ss.Latest(); latest != nil {
		fmt.Printf("  policy:  snapshot %s (%s)\n", latest.SnapshotID, latest.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
	} else {
		fmt.Printf("  policy:  no snapshot on disk\n")
	}
}

// runSnapshot handles snapshot management subcommands.
// Usage: emilyos snapshot capture
//        emilyos snapshot list
//        emilyos snapshot show <id>
func runSnapshot(args []string) {
	if len(args) == 0 {
		die("usage: emilyos snapshot capture|list|show <id>")
	}
	ss, err := policy.NewSnapshotStore(snapshotPath())
	if err != nil {
		die("snapshot store: %v", err)
	}
	switch args[0] {
	case "capture":
		ctx := callerContext()
		var prevID string
		if latest, _ := ss.Latest(); latest != nil {
			prevID = latest.SnapshotID
		}
		snap := &policy.Snapshot{
			SnapshotID:     policy.NewSnapshotID(),
			CreatedAt:      time.Now().UTC(),
			ActorID:        ctx.ActorID,
			GitCommit:      buildCommit,
			BuildID:        version + "+" + buildDate,
			PrevSnapshotID: prevID,
			Roles:          capturePolicyRoles(),
		}
		if err := ss.Write(snap); err != nil {
			die("write snapshot: %v", err)
		}
		fmt.Printf("snapshot captured: %s (hash=%s)\n", snap.SnapshotID, snap.Hash[:16])
	case "list":
		entries, err := os.ReadDir(snapshotPath())
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("no snapshots yet")
				return
			}
			die("list: %v", err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				fmt.Println(e.Name())
			}
		}
	case "show":
		if len(args) < 2 {
			die("snapshot show: <id> required")
		}
		snap, err := ss.Get(args[1])
		if err != nil {
			die("get: %v", err)
		}
		data, _ := json.MarshalIndent(snap, "", "  ")
		fmt.Println(string(data))
	default:
		die("unknown snapshot subcommand: %s", args[0])
	}
}

func capturePolicyRoles() map[string]any {
	out := make(map[string]any, len(policy.AllRoles()))
	for _, role := range policy.AllRoles() {
		caps := policy.CapsForRole(role)
		list := make([]string, 0, len(caps))
		for c := range caps {
			list = append(list, c)
		}
		out[role] = list
	}
	return out
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

func posturePath() string  { return getenvOr("EMILY_POSTURE_PATH", "var/posture.json") }
func auditPath() string    { return getenvOr("EMILY_AUDIT_PATH", "var/audit.jsonl") }
func snapshotPath() string { return getenvOr("EMILY_SNAPSHOT_DIR", "var/snapshots") }

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
