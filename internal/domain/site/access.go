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

type Membership struct {
	UserID string
	SiteID string
	Role   Role
}

type AccessDecision struct {
	ActorSite    string
	ResourceSite string
	Permission   Permission
	Role         Role
	Granted      bool
}

type RolePolicy struct {
	grants map[Role][]Permission
}

func NewRolePolicy() RolePolicy {
	return RolePolicy{grants: map[Role][]Permission{
		RoleAdministrator: {PermissionSiteRead, PermissionTelemetryWrite, PermissionRuleManage, PermissionAlertDispatch, PermissionMaintenance, PermissionReportExport},
		RoleSupervisor:    {PermissionSiteRead, PermissionRuleManage, PermissionAlertDispatch, PermissionReportExport},
		RoleDispatcher:    {PermissionSiteRead, PermissionAlertDispatch},
		RoleMaintainer:    {PermissionSiteRead, PermissionMaintenance},
		RoleViewer:        {PermissionSiteRead},
	}}
}

func (p RolePolicy) Decide(membership Membership, resourceSite string, permission Permission) AccessDecision {
	decision := AccessDecision{
		ActorSite:    membership.SiteID,
		ResourceSite: resourceSite,
		Permission:   permission,
		Role:         membership.Role,
	}
	for _, granted := range p.grants[membership.Role] {
		if granted == permission {
			decision.Granted = true
			break
		}
	}
	return decision
}

func (d AccessDecision) Error() error {
	if !d.Granted {
		return ErrAccessDenied
	}
	return nil
}

func (m Membership) Allows(permission Permission) bool {
	return NewRolePolicy().Decide(m, m.SiteID, permission).Granted
}

func Authorize(membership Membership, resourceSite string, permission Permission) error {
	return NewRolePolicy().Decide(membership, resourceSite, permission).Error()
}

func Require(membership Membership, resourceSite string, permission Permission) error {
	if membership.SiteID != resourceSite {
		return ErrAccessDenied
	}
	return Authorize(membership, resourceSite, permission)
}
