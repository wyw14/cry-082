package deviceapp

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

var ErrNotFound = errors.New("device not found")

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Repository interface {
	Save(context.Context, device.Device) error
	Find(context.Context, string) (device.Device, error)
	FindByCode(context.Context, string) (device.Device, error)
}
type AccessRepository interface {
	Membership(context.Context, string, string) (site.Membership, error)
}
type AuditRepository interface {
	AppendAudit(context.Context, audit.Entry) error
}

type Service struct {
	devices Repository
	access  AccessRepository
	audits  AuditRepository
	tx      transaction.Manager
	clock   Clock
	ids     IDGenerator
}

func New(devices Repository, access AccessRepository, audits AuditRepository, tx transaction.Manager, clock Clock, ids IDGenerator) *Service {
	return &Service{devices: devices, access: access, audits: audits, tx: tx, clock: clock, ids: ids}
}

type RegisterInput struct {
	Code, Model, SiteID, PointID, InstallLocation string
	Network                                       device.NetworkConfig
	ActorID, RequestID                            string
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (created device.Device, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	membership, err := s.access.Membership(ctx, input.ActorID, input.SiteID)
	if err != nil {
		return device.Device{}, err
	}
	if err := site.Require(membership, input.SiteID, site.PermissionMaintenance); err != nil {
		return device.Device{}, err
	}
	created, err = device.New(s.ids.NewID(), input.Code, input.Model, input.SiteID, input.PointID, input.InstallLocation, input.Network)
	if err != nil {
		return device.Device{}, err
	}
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if err := s.devices.Save(txctx, created); err != nil {
			return err
		}
		entry, err := audit.New(s.ids.NewID(), input.SiteID, input.ActorID, "api", "device.registered", "device", created.ID, "register monitoring device", input.RequestID, nil, map[string]string{"code": created.Code, "status": string(created.Status)}, s.clock.Now())
		if err != nil {
			return err
		}
		return s.audits.AppendAudit(txctx, entry)
	})
	return created, err
}

type TransitionInput struct {
	DeviceID                                  string
	Next                                      device.Status
	ReplacementID, ActorID, Reason, RequestID string
	ExpectedVersion                           int64
}

func (s *Service) Transition(ctx context.Context, input TransitionInput) (updated device.Device, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	current, err := s.devices.Find(ctx, input.DeviceID)
	if err != nil {
		return device.Device{}, err
	}
	membership, err := s.access.Membership(ctx, input.ActorID, current.SiteID)
	if err != nil {
		return device.Device{}, err
	}
	if err := site.Require(membership, current.SiteID, site.PermissionMaintenance); err != nil {
		return device.Device{}, err
	}
	if input.ExpectedVersion != current.Version {
		return device.Device{}, errors.New("device version conflict")
	}
	before := string(current.Status)
	if err := current.Transition(input.Next, s.clock.Now(), input.ReplacementID); err != nil {
		return device.Device{}, err
	}
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if err := s.devices.Save(txctx, current); err != nil {
			return err
		}
		entry, err := audit.New(s.ids.NewID(), current.SiteID, input.ActorID, "api", "device.transitioned", "device", current.ID, input.Reason, input.RequestID, map[string]string{"status": before}, map[string]string{"status": string(current.Status)}, s.clock.Now())
		if err != nil {
			return err
		}
		return s.audits.AppendAudit(txctx, entry)
	})
	return current, err
}
