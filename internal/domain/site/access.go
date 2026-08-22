package site

import "errors"

var ErrAccessDenied = errors.New("resource access denied")

type Role string

const (
	RoleAdministrator Role = "administrator"
	RoleSupervisor    Role = "supervisor"
	RoleDispatcher    Role = "dispatcher"
	RoleMaintainer    Role = "maintainer"
	RoleViewer        Role = "viewer"
)

type Permission string

const (
	PermissionSiteRead       Permission = "site:read"
	PermissionTelemetryWrite Permission = "telemetry:write"
	PermissionRuleManage     Permission = "rule:manage"
	PermissionAlertDispatch  Permission = "alert:dispatch"
	PermissionMaintenance    Permission = "maintenance:write"
	PermissionReportExport   Permission = "report:export"
)

var rolePermissions = map[Role]map[Permission]bool{
	RoleAdministrator: {PermissionSiteRead: true, PermissionTelemetryWrite: true, PermissionRuleManage: true, PermissionAlertDispatch: true, PermissionMaintenance: true, PermissionReportExport: true},
	RoleSupervisor:    {PermissionSiteRead: true, PermissionRuleManage: true, PermissionAlertDispatch: true, PermissionReportExport: true},
	RoleDispatcher:    {PermissionSiteRead: true, PermissionAlertDispatch: true},
	RoleMaintainer:    {PermissionSiteRead: true, PermissionMaintenance: true},
	RoleViewer:        {PermissionSiteRead: true},
}

type Membership struct {
	UserID string
	SiteID string
	Role   Role
}

func (m Membership) Allows(permission Permission) bool {
	allowed, ok := rolePermissions[m.Role]
	return ok && allowed[permission]
}

func Require(m Membership, siteID string, permission Permission) error {
	if m.SiteID != siteID || !m.Allows(permission) {
		return ErrAccessDenied
	}
	return nil
}
