// Package policy implements EmilyOS RBAC — roles, capabilities, and policy snapshots.
//
// Three roles are defined and fixed in v0 (simplicity = auditability):
//
//	Operator — normal use
//	Admin    — policy changes, export, audit read
//	Auditor  — read-only audit log access
//
// Capabilities are named strings of the form "cap.*". Role capability sets are
// defined here. Additional capabilities can be granted per-identity via the
// identity override map (not yet implemented in v0 — reserved for Milestone 4).
package policy

// Role names.
const (
	RoleOperator = "operator"
	RoleAdmin    = "admin"
	RoleAuditor  = "auditor"
)

// Capability names. All verb capability checks use these constants.
const (
	CapSessionOpen    = "cap.session.open"
	CapPostureSet     = "cap.posture.set"
	CapPostureAdmin   = "cap.posture.admin" // required for INCIDENT posture
	CapExec           = "cap.exec"
	CapNet            = "cap.net"
	CapDomainStart    = "cap.domain.start"
	CapDomainStop     = "cap.domain.stop"
	CapSSHConnect     = "cap.ssh.connect"
	CapSSHManageHosts = "cap.ssh.manage_hosts"
	CapSSHManageKeys  = "cap.ssh.manage_keys"
	CapPolicyWrite    = "cap.policy.write"
	CapAuditRead      = "cap.audit.read"
	CapExport         = "cap.export"
)

// roleCaps defines the fixed capability set for each role.
var roleCaps = map[string]map[string]bool{
	RoleOperator: {
		CapSessionOpen: true,
		CapPostureSet:  true,
		CapExec:        true,
		CapNet:         true,
		CapDomainStart: true,
		CapDomainStop:  true,
		CapSSHConnect:  true,
	},
	RoleAdmin: {
		CapSessionOpen:    true,
		CapPostureSet:     true,
		CapPostureAdmin:   true,
		CapExec:           true,
		CapNet:            true,
		CapDomainStart:    true,
		CapDomainStop:     true,
		CapSSHConnect:     true,
		CapSSHManageHosts: true,
		CapSSHManageKeys:  true,
		CapPolicyWrite:    true,
		CapAuditRead:      true,
		CapExport:         true,
	},
	RoleAuditor: {
		CapAuditRead: true,
		CapExport:    true,
	},
}

// HasCapability returns true if the given role has the given capability.
func HasCapability(role, cap string) bool {
	caps, ok := roleCaps[role]
	if !ok {
		return false
	}
	return caps[cap]
}

// CapsForRole returns a copy of the capability set for the given role.
func CapsForRole(role string) map[string]bool {
	caps := make(map[string]bool)
	for k, v := range roleCaps[role] {
		caps[k] = v
	}
	return caps
}

// ValidRole returns true if role is a known role.
func ValidRole(role string) bool {
	_, ok := roleCaps[role]
	return ok
}

// AllRoles returns the list of all defined roles.
func AllRoles() []string {
	return []string{RoleOperator, RoleAdmin, RoleAuditor}
}
