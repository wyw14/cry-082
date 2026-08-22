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
	deviceEntity, err := s.authorizeRecord(ctx, input)
	if err != nil {
		return maintenance.Record{}, err
	}
	record, err = s.buildRecord(input)
	if err != nil {
		return maintenance.Record{}, err
	}
	if err := s.maintenance.SaveRecord(ctx, record); err != nil {
		return maintenance.Record{}, err
	}
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if err := s.applyDeviceChange(txctx, &deviceEntity, input); err != nil {
			return err
		}
		return s.appendRecordAudit(txctx, deviceEntity.SiteID, record, input)
	})
	return record, err
}

func (s *Service) authorizeRecord(ctx context.Context, input RecordInput) (device.Device, error) {
	deviceEntity, err := s.devices.Find(ctx, input.DeviceID)
	if err != nil {
		return device.Device{}, err
	}
	membership, err := s.access.Membership(ctx, input.ActorID, deviceEntity.SiteID)
	if err != nil {
		return device.Device{}, err
	}
	if err := site.Require(membership, deviceEntity.SiteID, site.PermissionMaintenance); err != nil {
		return device.Device{}, err
	}
	return deviceEntity, nil
}

func (s *Service) buildRecord(input RecordInput) (maintenance.Record, error) {
	return maintenance.NewRecord(
		s.ids.NewID(),
		input.DeviceID,
		input.ActorID,
		input.Type,
		input.StartedAt,
		input.CompletedAt,
		input.Reason,
		input.Result,
		input.ReplacementID,
		input.AttachmentIDs,
	)
}

func (s *Service) applyDeviceChange(ctx context.Context, deviceEntity *device.Device, input RecordInput) error {
	if input.Type != maintenance.TypeReplacement {
		return nil
	}
	if err := deviceEntity.Transition(device.StatusReplaced, input.CompletedAt, input.ReplacementID); err != nil {
		return err
	}
	return s.devices.Save(ctx, *deviceEntity)
}

func (s *Service) appendRecordAudit(ctx context.Context, siteID string, record maintenance.Record, input RecordInput) error {
	entry, err := audit.New(
		s.ids.NewID(),
		siteID,
		input.ActorID,
		"api",
		"maintenance.recorded",
		"device",
		input.DeviceID,
		input.Reason,
		input.RequestID,
		nil,
		map[string]string{"type": string(record.Type), "result": record.Result},
		s.clock.Now(),
	)
	if err != nil {
		return err
	}
	return s.audits.AppendAudit(ctx, entry)
}
