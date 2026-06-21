package policy_test

import (
	"testing"

	. "emilyos/internal/policy"
)

func TestHasCapability_OperatorCaps(t *testing.T) {
	cases := []struct {
		cap    string
		expect bool
	}{
		{CapSessionOpen, true},
		{CapPostureSet, true},
		{CapExec, true},
		{CapNet, true},
		{CapDomainStart, true},
		{CapDomainStop, true},
		{CapSSHConnect, true},
		// Operator must NOT have admin caps
		{CapPostureAdmin, false},
		{CapPolicyWrite, false},
		{CapAuditRead, false},
		{CapExport, false},
		{CapSSHManageHosts, false},
		{CapSSHManageKeys, false},
	}
	for _, tc := range cases {
		got := HasCapability(RoleOperator, tc.cap)
		if got != tc.expect {
			t.Errorf("Operator.%s = %v, want %v", tc.cap, got, tc.expect)
		}
	}
}

func TestHasCapability_AuditorOnlyReads(t *testing.T) {
	allowed := []string{CapAuditRead, CapExport}
	forbidden := []string{
		CapSessionOpen, CapPostureSet, CapPostureAdmin,
		CapExec, CapNet, CapDomainStart, CapDomainStop,
		CapSSHConnect, CapSSHManageHosts, CapSSHManageKeys, CapPolicyWrite,
	}
	for _, cap := range allowed {
		if !HasCapability(RoleAuditor, cap) {
			t.Errorf("Auditor should have %s", cap)
		}
	}
	for _, cap := range forbidden {
		if HasCapability(RoleAuditor, cap) {
			t.Errorf("Auditor must NOT have %s", cap)
		}
	}
}

func TestHasCapability_AdminHasAll(t *testing.T) {
	all := []string{
		CapSessionOpen, CapPostureSet, CapPostureAdmin, CapExec, CapNet,
		CapDomainStart, CapDomainStop, CapSSHConnect, CapSSHManageHosts,
		CapSSHManageKeys, CapPolicyWrite, CapAuditRead, CapExport,
	}
	for _, cap := range all {
		if !HasCapability(RoleAdmin, cap) {
			t.Errorf("Admin should have %s", cap)
		}
	}
}

func TestHasCapability_UnknownRole(t *testing.T) {
	if HasCapability("superuser", CapExec) {
		t.Error("unknown role must not have any capability")
	}
}

func TestHasCapability_UnknownCap(t *testing.T) {
	if HasCapability(RoleAdmin, "cap.unknown") {
		t.Error("no role should have an unknown capability")
	}
}

func TestCapsForRole_ReturnsCopy(t *testing.T) {
	caps := CapsForRole(RoleOperator)
	caps[CapPolicyWrite] = true // mutate the copy
	// Original must be unchanged
	if HasCapability(RoleOperator, CapPolicyWrite) {
		t.Error("CapsForRole must return a copy, not a reference to the internal map")
	}
}

func TestValidRole(t *testing.T) {
	cases := []struct {
		role   string
		expect bool
	}{
		{RoleOperator, true},
		{RoleAdmin, true},
		{RoleAuditor, true},
		{"superuser", false},
		{"", false},
	}
	for _, tc := range cases {
		if ValidRole(tc.role) != tc.expect {
			t.Errorf("ValidRole(%q) = %v, want %v", tc.role, !tc.expect, tc.expect)
		}
	}
}

func TestAllRoles(t *testing.T) {
	roles := AllRoles()
	if len(roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(roles))
	}
	seen := make(map[string]bool)
	for _, r := range roles {
		seen[r] = true
	}
	for _, want := range []string{RoleOperator, RoleAdmin, RoleAuditor} {
		if !seen[want] {
			t.Errorf("AllRoles() missing %q", want)
		}
	}
}
