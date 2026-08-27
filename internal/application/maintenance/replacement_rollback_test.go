package maintenanceapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/maintenance"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

type maintenanceTransactionKey struct{}

type maintenanceRecordRepository struct {
	committed []maintenance.Record
}

func (r *maintenanceRecordRepository) SaveRecord(ctx context.Context, value maintenance.Record) error {
	if unit, ok := ctx.Value(maintenanceTransactionKey{}).(*maintenanceUnit); ok {
		unit.pending = append(unit.pending, value)
		return nil
	}
	r.committed = append(r.committed, value)
	return nil
}
func (*maintenanceRecordRepository) SaveCalibration(context.Context, maintenance.Calibration) error {
	return nil
}

type maintenanceUnit struct {
	repository *maintenanceRecordRepository
	pending    []maintenance.Record
}

func (u *maintenanceUnit) Bind(ctx context.Context) context.Context {
	return context.WithValue(ctx, maintenanceTransactionKey{}, u)
}
func (u *maintenanceUnit) Commit(context.Context) error {
	u.repository.committed = append(u.repository.committed, u.pending...)
	return nil
}
func (u *maintenanceUnit) Rollback(context.Context) error {
	u.pending = nil
	return nil
}

type maintenanceTransactionManager struct{ repository *maintenanceRecordRepository }

func (m maintenanceTransactionManager) Begin(context.Context) (transaction.Unit, error) {
	return &maintenanceUnit{repository: m.repository}, nil
}

type failingReplacementDeviceRepository struct{ value device.Device }

func (r failingReplacementDeviceRepository) Find(context.Context, string) (device.Device, error) {
	return r.value, nil
}
func (failingReplacementDeviceRepository) Save(context.Context, device.Device) error {
	return errors.New("device write failed")
}

type maintenanceAccessRepository struct{}

func (maintenanceAccessRepository) Membership(context.Context, string, string) (site.Membership, error) {
	return site.Membership{UserID: "maintainer", SiteID: "site", Role: site.RoleMaintainer}, nil
}

type maintenanceAuditRepository struct{}

func (maintenanceAuditRepository) AppendAudit(context.Context, audit.Entry) error { return nil }

type maintenanceClock struct{ value time.Time }

func (c maintenanceClock) Now() time.Time { return c.value }

type maintenanceIDs struct{}

func (maintenanceIDs) NewID() string { return "maintenance-record" }

func TestFailedReplacementDoesNotLeaveMaintenanceHistory(t *testing.T) {
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	records := &maintenanceRecordRepository{}
	devices := failingReplacementDeviceRepository{value: device.Device{ID: "device", SiteID: "site", Status: device.StatusOnline, Version: 1}}
	service := New(records, devices, maintenanceAccessRepository{}, maintenanceAuditRepository{}, maintenanceTransactionManager{repository: records}, maintenanceClock{value: now}, maintenanceIDs{})
	_, err := service.Record(context.Background(), RecordInput{DeviceID: "device", Type: maintenance.TypeReplacement, ActorID: "maintainer", Reason: "replace damaged sensor", Result: "replacement attempted", ReplacementID: "replacement", RequestID: "request-id", StartedAt: now.Add(-time.Hour), CompletedAt: now})
	if err == nil {
		t.Fatal("replacement unexpectedly succeeded")
	}
	if len(records.committed) != 0 {
		t.Fatalf("failed replacement left maintenance history: %+v", records.committed)
	}
}
