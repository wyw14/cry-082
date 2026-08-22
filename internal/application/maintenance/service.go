package maintenanceapp

import (
	"context"
	"time"

	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/maintenance"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Repository interface {
	SaveRecord(context.Context, maintenance.Record) error
	SaveCalibration(context.Context, maintenance.Calibration) error
}
type DeviceRepository interface {
	Find(context.Context, string) (device.Device, error)
	Save(context.Context, device.Device) error
}
type AccessRepository interface {
	Membership(context.Context, string, string) (site.Membership, error)
}
type AuditRepository interface {
	AppendAudit(context.Context, audit.Entry) error
}

type Service struct {
	maintenance Repository
	devices     DeviceRepository
	access      AccessRepository
	audits      AuditRepository
	tx          transaction.Manager
	clock       Clock
	ids         IDGenerator
}

func New(repository Repository, devices DeviceRepository, access AccessRepository, audits AuditRepository, tx transaction.Manager, clock Clock, ids IDGenerator) *Service {
	return &Service{maintenance: repository, devices: devices, access: access, audits: audits, tx: tx, clock: clock, ids: ids}
}

type RecordInput struct {
	DeviceID                                          string
	Type                                              maintenance.Type
	ActorID, Reason, Result, ReplacementID, RequestID string
	StartedAt, CompletedAt                            time.Time
	AttachmentIDs                                     []string
}

func (s *Service) Record(ctx context.Context, input RecordInput) (record maintenance.Record, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	deviceEntity, err := s.devices.Find(ctx, input.DeviceID)
	if err != nil {
		return maintenance.Record{}, err
	}
	membership, err := s.access.Membership(ctx, input.ActorID, deviceEntity.SiteID)
	if err != nil {
		return maintenance.Record{}, err
	}
	if err := site.Require(membership, deviceEntity.SiteID, site.PermissionMaintenance); err != nil {
		return maintenance.Record{}, err
	}
	record, err = maintenance.NewRecord(s.ids.NewID(), input.DeviceID, input.ActorID, input.Type, input.StartedAt, input.CompletedAt, input.Reason, input.Result, input.ReplacementID, input.AttachmentIDs)
	if err != nil {
		return maintenance.Record{}, err
	}
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if input.Type == maintenance.TypeReplacement {
			if err := deviceEntity.Transition(device.StatusReplaced, input.CompletedAt, input.ReplacementID); err != nil {
				return err
			}
			if err := s.devices.Save(txctx, deviceEntity); err != nil {
				return err
			}
		}
		if err := s.maintenance.SaveRecord(txctx, record); err != nil {
			return err
		}
		entry, err := audit.New(s.ids.NewID(), deviceEntity.SiteID, input.ActorID, "api", "maintenance.recorded", "device", input.DeviceID, input.Reason, input.RequestID, nil, map[string]string{"type": string(input.Type), "result": input.Result}, s.clock.Now())
		if err != nil {
			return err
		}
		return s.audits.AppendAudit(txctx, entry)
	})
	return record, err
}
