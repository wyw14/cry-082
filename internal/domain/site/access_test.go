package site

import (
	"errors"
	"testing"
)

func TestRoleAndResourceOwnershipAreBothRequired(t *testing.T) {
	supervisor := Membership{UserID: "u1", SiteID: "s1", Role: RoleSupervisor}
	if err := Require(supervisor, "s1", PermissionRuleManage); err != nil {
		t.Fatal(err)
	}
	if err := Require(supervisor, "s2", PermissionRuleManage); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("cross-site access=%v", err)
	}
	viewer := Membership{UserID: "u2", SiteID: "s1", Role: RoleViewer}
	if err := Require(viewer, "s1", PermissionAlertDispatch); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("viewer dispatch=%v", err)
	}
}
