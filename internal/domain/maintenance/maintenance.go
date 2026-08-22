package maintenance

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidMaintenance = errors.New("invalid maintenance record")

type Type string

const (
	TypeInspection  Type = "inspection"
	TypeRepair      Type = "repair"
	TypeCalibration Type = "calibration"
	TypeReplacement Type = "replacement"
)

type Record struct {
	ID            string
	DeviceID      string
	Type          Type
	PerformedBy   string
	StartedAt     time.Time
	CompletedAt   time.Time
	Reason        string
	Result        string
	ReplacementID string
	AttachmentIDs []string
}

func NewRecord(id, deviceID, actor string, kind Type, startedAt, completedAt time.Time, reason, result, replacementID string, attachments []string) (Record, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" || strings.TrimSpace(result) == "" || completedAt.Before(startedAt) {
		return Record{}, ErrInvalidMaintenance
	}
	if kind == TypeReplacement && strings.TrimSpace(replacementID) == "" {
		return Record{}, ErrInvalidMaintenance
	}
	return Record{ID: id, DeviceID: deviceID, Type: kind, PerformedBy: actor, StartedAt: startedAt.UTC(), CompletedAt: completedAt.UTC(), Reason: reason, Result: result, ReplacementID: replacementID, AttachmentIDs: append([]string(nil), attachments...)}, nil
}

type Calibration struct {
	ID             string
	DeviceID       string
	SchemaID       string
	ReferenceValue float64
	ObservedValue  float64
	Offset         float64
	PerformedBy    string
	PerformedAt    time.Time
	ExpiresAt      time.Time
	CertificateID  string
}

func NewCalibration(id, deviceID, schemaID, actor string, reference, observed float64, at, expiresAt time.Time, certificateID string) (Calibration, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(schemaID) == "" || strings.TrimSpace(actor) == "" || !at.Before(expiresAt) {
		return Calibration{}, ErrInvalidMaintenance
	}
	return Calibration{ID: id, DeviceID: deviceID, SchemaID: schemaID, ReferenceValue: reference, ObservedValue: observed, Offset: reference - observed, PerformedBy: actor, PerformedAt: at.UTC(), ExpiresAt: expiresAt.UTC(), CertificateID: certificateID}, nil
}
