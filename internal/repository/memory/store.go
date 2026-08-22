package memory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	alertapp "github.com/wyw14/cry-082/internal/application/alerts"
	"github.com/wyw14/cry-082/internal/domain/alert"
	"github.com/wyw14/cry-082/internal/domain/artifact"
	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/auth"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/maintenance"
	"github.com/wyw14/cry-082/internal/domain/report"
	"github.com/wyw14/cry-082/internal/domain/rule"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
)

var ErrNotFound = errors.New("record not found")

type Store struct {
	topology   topologyState
	equipment  equipmentState
	telemetry  telemetryState
	policy     policyState
	incidents  incidentState
	operations operationsState
	reporting  reportingState
	identity   identityState
	artifacts  artifactState
}

type topologyState struct {
	sync.RWMutex
	sites       map[string]site.Site
	zones       map[string]site.Zone
	points      map[string]site.MonitoringPoint
	memberships map[string]site.Membership
}

type equipmentState struct {
	sync.RWMutex
	devices     map[string]device.Device
	deviceCodes map[string]string
}

type telemetryState struct {
	sync.RWMutex
	schemas               map[string]telemetry.Schema
	observations          map[string]telemetry.Observation
	observationIdentities map[string]string
}

type policyState struct {
	sync.RWMutex
	rules          map[string][]rule.Version
	evaluations    []rule.Evaluation
	recalculations map[string]rule.Recalculation
}

type incidentState struct {
	sync.RWMutex
	alerts     map[string]alert.Alert
	workOrders map[string]alert.WorkOrder
}

type operationsState struct {
	sync.RWMutex
	maintenanceRecords map[string]maintenance.Record
	calibrations       map[string]maintenance.Calibration
	audits             []audit.Entry
}

type reportingState struct {
	sync.RWMutex
	reports map[string]report.DailyReport
	exports map[string]report.Export
}

type identityState struct {
	sync.RWMutex
	users         map[string]auth.User
	usernames     map[string]string
	refreshTokens map[string]auth.RefreshToken
}

type artifactState struct {
	sync.RWMutex
	files map[string]artifact.File
}

func New() *Store {
	return &Store{
		topology: topologyState{
			sites:       make(map[string]site.Site),
			zones:       make(map[string]site.Zone),
			points:      make(map[string]site.MonitoringPoint),
			memberships: make(map[string]site.Membership),
		},
		equipment: equipmentState{
			devices:     make(map[string]device.Device),
			deviceCodes: make(map[string]string),
		},
		telemetry: telemetryState{
			schemas:               make(map[string]telemetry.Schema),
			observations:          make(map[string]telemetry.Observation),
			observationIdentities: make(map[string]string),
		},
		policy: policyState{
			rules:          make(map[string][]rule.Version),
			recalculations: make(map[string]rule.Recalculation),
		},
		incidents: incidentState{
			alerts:     make(map[string]alert.Alert),
			workOrders: make(map[string]alert.WorkOrder),
		},
		operations: operationsState{
			maintenanceRecords: make(map[string]maintenance.Record),
			calibrations:       make(map[string]maintenance.Calibration),
		},
		reporting: reportingState{
			reports: make(map[string]report.DailyReport),
			exports: make(map[string]report.Export),
		},
		identity: identityState{
			users:         make(map[string]auth.User),
			usernames:     make(map[string]string),
			refreshTokens: make(map[string]auth.RefreshToken),
		},
		artifacts: artifactState{files: make(map[string]artifact.File)},
	}
}

func membershipKey(userID, siteID string) string { return userID + "\x00" + siteID }
func (s *Store) SaveSite(ctx context.Context, value site.Site) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.topology.Lock()
	defer s.topology.Unlock()
	s.topology.sites[value.ID] = value
	return nil
}
func (s *Store) SaveZone(ctx context.Context, value site.Zone) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.topology.Lock()
	defer s.topology.Unlock()
	if _, ok := s.topology.sites[value.SiteID]; !ok {
		return ErrNotFound
	}
	s.topology.zones[value.ID] = value
	return nil
}
func (s *Store) SavePoint(ctx context.Context, value site.MonitoringPoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.topology.Lock()
	defer s.topology.Unlock()
	if _, ok := s.topology.zones[value.ZoneID]; !ok {
		return ErrNotFound
	}
	s.topology.points[value.ID] = value
	return nil
}
func (s *Store) SaveMembership(ctx context.Context, value site.Membership) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.topology.Lock()
	defer s.topology.Unlock()
	s.topology.memberships[membershipKey(value.UserID, value.SiteID)] = value
	return nil
}
func (s *Store) Membership(ctx context.Context, userID, siteID string) (site.Membership, error) {
	if err := ctx.Err(); err != nil {
		return site.Membership{}, err
	}
	s.topology.RLock()
	defer s.topology.RUnlock()
	value, ok := s.topology.memberships[membershipKey(userID, siteID)]
	if !ok {
		return site.Membership{}, ErrNotFound
	}
	return value, nil
}

func (s *Store) Save(ctx context.Context, value device.Device) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.equipment.Lock()
	defer s.equipment.Unlock()
	if existingID, ok := s.equipment.deviceCodes[value.Code]; ok && existingID != value.ID {
		return errors.New("device code already exists")
	}
	if existing, ok := s.equipment.devices[value.ID]; ok && value.Version < existing.Version {
		return errors.New("device version conflict")
	}
	s.equipment.devices[value.ID] = value
	s.equipment.deviceCodes[value.Code] = value.ID
	return nil
}
func (s *Store) Find(ctx context.Context, id string) (device.Device, error) {
	if err := ctx.Err(); err != nil {
		return device.Device{}, err
	}
	s.equipment.RLock()
	defer s.equipment.RUnlock()
	value, ok := s.equipment.devices[id]
	if !ok {
		return device.Device{}, ErrNotFound
	}
	return value, nil
}
func (s *Store) FindByCode(ctx context.Context, code string) (device.Device, error) {
	s.equipment.RLock()
	id, ok := s.equipment.deviceCodes[code]
	s.equipment.RUnlock()
	if !ok {
		return device.Device{}, ErrNotFound
	}
	return s.Find(ctx, id)
}

func (s *Store) SaveSchema(ctx context.Context, value telemetry.Schema) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.telemetry.Lock()
	defer s.telemetry.Unlock()
	s.telemetry.schemas[value.ID] = value
	return nil
}
func (s *Store) FindSchema(ctx context.Context, id string) (telemetry.Schema, error) {
	if err := ctx.Err(); err != nil {
		return telemetry.Schema{}, err
	}
	s.telemetry.RLock()
	defer s.telemetry.RUnlock()
	value, ok := s.telemetry.schemas[id]
	if !ok {
		return telemetry.Schema{}, ErrNotFound
	}
	return value, nil
}
func (s *Store) ExistsIdentity(ctx context.Context, identity string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.telemetry.RLock()
	defer s.telemetry.RUnlock()
	_, ok := s.telemetry.observationIdentities[identity]
	return ok, nil
}
func (s *Store) Latest(ctx context.Context, deviceID, schemaID string) (*telemetry.Observation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.telemetry.RLock()
	defer s.telemetry.RUnlock()
	var latest *telemetry.Observation
	for _, value := range s.telemetry.observations {
		if value.DeviceID != deviceID || value.SchemaID != schemaID {
			continue
		}
		candidate := value
		if latest == nil || candidate.SampledAt.After(latest.SampledAt) {
			latest = &candidate
		}
	}
	return latest, nil
}
func (s *Store) Append(ctx context.Context, value telemetry.Observation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.telemetry.Lock()
	defer s.telemetry.Unlock()
	if _, ok := s.telemetry.observations[value.ID]; ok {
		return errors.New("observation already exists")
	}
	if _, ok := s.telemetry.observationIdentities[value.IdempotencyKey]; ok {
		return errors.New("observation identity already exists")
	}
	s.telemetry.observations[value.ID] = value
	s.telemetry.observationIdentities[value.IdempotencyKey] = value.ID
	return nil
}
func (s *Store) FindObservation(ctx context.Context, id string) (telemetry.Observation, error) {
	if err := ctx.Err(); err != nil {
		return telemetry.Observation{}, err
	}
	s.telemetry.RLock()
	defer s.telemetry.RUnlock()
	value, ok := s.telemetry.observations[id]
	if !ok {
		return telemetry.Observation{}, ErrNotFound
	}
	return value, nil
}
func (s *Store) Range(ctx context.Context, siteID string, start, end time.Time) ([]telemetry.Observation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.telemetry.RLock()
	defer s.telemetry.RUnlock()
	result := make([]telemetry.Observation, 0)
	for _, value := range s.telemetry.observations {
		if value.SiteID == siteID && !value.SampledAt.Before(start) && value.SampledAt.Before(end) {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SampledAt.Before(result[j].SampledAt) })
	return result, nil
}

func (s *Store) LatestRule(ctx context.Context, ruleID string) (*rule.Version, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.policy.RLock()
	defer s.policy.RUnlock()
	versions := s.policy.rules[ruleID]
	if len(versions) == 0 {
		return nil, nil
	}
	value := versions[len(versions)-1]
	return &value, nil
}
func (s *Store) SaveRule(ctx context.Context, value rule.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.policy.Lock()
	defer s.policy.Unlock()
	versions := s.policy.rules[value.RuleID]
	replaced := false
	for index, existing := range versions {
		if existing.Version == value.Version {
			versions[index] = value
			replaced = true
			break
		}
	}
	if !replaced {
		versions = append(versions, value)
		sort.Slice(versions, func(i, j int) bool { return versions[i].Version < versions[j].Version })
	}
	s.policy.rules[value.RuleID] = versions
	return nil
}
func (s *Store) ActiveForSite(ctx context.Context, siteID string, at time.Time) ([]rule.Version, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.policy.RLock()
	defer s.policy.RUnlock()
	result := make([]rule.Version, 0)
	for _, versions := range s.policy.rules {
		for _, value := range versions {
			if value.SiteID == siteID && value.Status == rule.StatusActive && !value.EffectiveFrom.After(at) {
				result = append(result, value)
			}
		}
	}
	return result, nil
}
func (s *Store) AppendEvaluation(ctx context.Context, value rule.Evaluation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.policy.Lock()
	defer s.policy.Unlock()
	s.policy.evaluations = append(s.policy.evaluations, value)
	return nil
}
func (s *Store) SaveRecalculation(ctx context.Context, value rule.Recalculation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.policy.Lock()
	defer s.policy.Unlock()
	s.policy.recalculations[value.ID] = value
	return nil
}

func (s *Store) FindMergeable(ctx context.Context, mergeKey string, kind alert.Kind, since time.Time) (*alert.Alert, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.incidents.RLock()
	defer s.incidents.RUnlock()
	for _, value := range s.incidents.alerts {
		if value.MergeKey == mergeKey && value.Kind == kind && !value.LastSignalAt.Before(since) && value.Status != alert.StatusClosed {
			candidate := value
			return &candidate, nil
		}
	}
	return nil, nil
}
func (s *Store) SaveAlert(ctx context.Context, value alert.Alert) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.incidents.Lock()
	defer s.incidents.Unlock()
	s.incidents.alerts[value.ID] = value
	return nil
}
func (s *Store) FindAlert(ctx context.Context, id string) (alert.Alert, error) {
	if err := ctx.Err(); err != nil {
		return alert.Alert{}, err
	}
	s.incidents.RLock()
	defer s.incidents.RUnlock()
	value, ok := s.incidents.alerts[id]
	if !ok {
		return alert.Alert{}, ErrNotFound
	}
	return value, nil
}
func (s *Store) RangeAlerts(ctx context.Context, siteID string, start, end time.Time) ([]alert.Alert, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.incidents.RLock()
	defer s.incidents.RUnlock()
	result := make([]alert.Alert, 0)
	for _, value := range s.incidents.alerts {
		if value.SiteID == siteID && !value.StartedAt.Before(start) && value.StartedAt.Before(end) {
			result = append(result, value)
		}
	}
	return result, nil
}
func (s *Store) ListAlerts(ctx context.Context, filter alertapp.AlertFilter) ([]alert.Alert, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	s.incidents.RLock()
	defer s.incidents.RUnlock()
	result := make([]alert.Alert, 0)
	for _, value := range s.incidents.alerts {
		if value.SiteID != filter.SiteID || (filter.Kind != "" && value.Kind != filter.Kind) || (filter.Status != "" && value.Status != filter.Status) {
			continue
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		var less bool
		switch filter.Sort {
		case "last_signal_at":
			less = result[i].LastSignalAt.Before(result[j].LastSignalAt)
		case "status":
			less = result[i].Status < result[j].Status
		default:
			less = result[i].StartedAt.Before(result[j].StartedAt)
		}
		if filter.Descending {
			return !less
		}
		return less
	})
	total := len(result)
	if filter.Offset >= total {
		return []alert.Alert{}, total, nil
	}
	end := filter.Offset + filter.Limit
	if end > total {
		end = total
	}
	return append([]alert.Alert(nil), result[filter.Offset:end]...), total, nil
}
func (s *Store) SaveWorkOrder(ctx context.Context, value alert.WorkOrder) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.incidents.Lock()
	defer s.incidents.Unlock()
	s.incidents.workOrders[value.ID] = value
	return nil
}
func (s *Store) SaveRecord(ctx context.Context, value maintenance.Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.operations.Lock()
	defer s.operations.Unlock()
	s.operations.maintenanceRecords[value.ID] = value
	return nil
}
func (s *Store) SaveCalibration(ctx context.Context, value maintenance.Calibration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.operations.Lock()
	defer s.operations.Unlock()
	s.operations.calibrations[value.ID] = value
	return nil
}
func (s *Store) AppendAudit(ctx context.Context, value audit.Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.operations.Lock()
	defer s.operations.Unlock()
	s.operations.audits = append(s.operations.audits, value)
	return nil
}
func (s *Store) Audits(siteID string) []audit.Entry {
	s.operations.RLock()
	defer s.operations.RUnlock()
	result := make([]audit.Entry, 0)
	for _, value := range s.operations.audits {
		if value.SiteID == siteID {
			result = append(result, value)
		}
	}
	return result
}

func (s *Store) SaveDaily(ctx context.Context, value report.DailyReport) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.reporting.Lock()
	defer s.reporting.Unlock()
	s.reporting.reports[value.ID] = value
	return nil
}
func (s *Store) FindDaily(ctx context.Context, id string) (report.DailyReport, error) {
	if err := ctx.Err(); err != nil {
		return report.DailyReport{}, err
	}
	s.reporting.RLock()
	defer s.reporting.RUnlock()
	value, ok := s.reporting.reports[id]
	if !ok {
		return report.DailyReport{}, ErrNotFound
	}
	return value, nil
}
func (s *Store) SaveExport(ctx context.Context, value report.Export) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.reporting.Lock()
	defer s.reporting.Unlock()
	s.reporting.exports[value.ID] = value
	return nil
}

func (s *Store) SaveUser(ctx context.Context, value auth.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.identity.Lock()
	defer s.identity.Unlock()
	s.identity.users[value.ID] = value
	s.identity.usernames[strings.ToLower(value.Username)] = value.ID
	return nil
}
func (s *Store) FindUserByUsername(ctx context.Context, username string) (auth.User, error) {
	if err := ctx.Err(); err != nil {
		return auth.User{}, err
	}
	s.identity.RLock()
	defer s.identity.RUnlock()
	id, ok := s.identity.usernames[strings.ToLower(username)]
	if !ok {
		if value, exists := s.identity.users[username]; exists {
			return value, nil
		}
		return auth.User{}, ErrNotFound
	}
	return s.identity.users[id], nil
}
func (s *Store) FindUserByID(ctx context.Context, id string) (auth.User, error) {
	if err := ctx.Err(); err != nil {
		return auth.User{}, err
	}
	s.identity.RLock()
	defer s.identity.RUnlock()
	value, ok := s.identity.users[id]
	if !ok {
		return auth.User{}, ErrNotFound
	}
	return value, nil
}
func (s *Store) SaveRefreshToken(ctx context.Context, value auth.RefreshToken) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.identity.Lock()
	defer s.identity.Unlock()
	s.identity.refreshTokens[value.ID] = value
	return nil
}
func (s *Store) FindRefreshToken(ctx context.Context, id string) (auth.RefreshToken, error) {
	if err := ctx.Err(); err != nil {
		return auth.RefreshToken{}, err
	}
	s.identity.RLock()
	defer s.identity.RUnlock()
	value, ok := s.identity.refreshTokens[id]
	if !ok {
		return auth.RefreshToken{}, ErrNotFound
	}
	return value, nil
}

func (s *Store) RotateRefreshToken(ctx context.Context, current, replacement auth.RefreshToken) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.identity.Lock()
	defer s.identity.Unlock()
	stored, ok := s.identity.refreshTokens[current.ID]
	if !ok || stored.RevokedAt != nil || stored.Digest != current.Digest {
		return auth.ErrInvalidToken
	}
	if _, exists := s.identity.refreshTokens[replacement.ID]; exists {
		return auth.ErrInvalidToken
	}
	s.identity.refreshTokens[current.ID] = current
	s.identity.refreshTokens[replacement.ID] = replacement
	return nil
}

func (s *Store) SaveFile(ctx context.Context, value artifact.File) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.artifacts.Lock()
	defer s.artifacts.Unlock()
	if _, exists := s.artifacts.files[value.ID]; exists {
		return errors.New("file already exists")
	}
	s.artifacts.files[value.ID] = value
	return nil
}

func (s *Store) FindFile(ctx context.Context, id string) (artifact.File, error) {
	if err := ctx.Err(); err != nil {
		return artifact.File{}, err
	}
	s.artifacts.RLock()
	defer s.artifacts.RUnlock()
	value, ok := s.artifacts.files[id]
	if !ok {
		return artifact.File{}, ErrNotFound
	}
	return value, nil
}
