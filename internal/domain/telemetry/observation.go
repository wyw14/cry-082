package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidObservation = errors.New("invalid observation")
	ErrCorrectionTarget   = errors.New("invalid correction target")
)

type Quality string

const (
	QualityAccepted    Quality = "accepted"
	QualitySuspect     Quality = "suspect"
	QualityQuarantined Quality = "quarantined"
)

type Observation struct {
	ID             string
	DeviceID       string
	SiteID         string
	PointID        string
	SchemaID       string
	Metric         Metric
	Value          float64
	Unit           string
	SampledAt      time.Time
	ReceivedAt     time.Time
	CorrectedAt    time.Time
	CorrectionOf   string
	Quality        Quality
	QualityReasons []string
	IdempotencyKey string
	SourceBatchID  string
}

func Identity(deviceID, schemaID string, sampledAt time.Time) string {
	raw := fmt.Sprintf("%s\x00%s\x00%s", strings.TrimSpace(deviceID), strings.TrimSpace(schemaID), sampledAt.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func NewObservation(id, deviceID, siteID, pointID string, schema Schema, value float64, sampledAt, receivedAt time.Time, batchID string) (Observation, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(pointID) == "" || sampledAt.IsZero() || receivedAt.IsZero() {
		return Observation{}, ErrInvalidObservation
	}
	sampledAt = sampledAt.UTC()
	receivedAt = receivedAt.UTC()
	return Observation{
		ID: id, DeviceID: deviceID, SiteID: siteID, PointID: pointID, SchemaID: schema.ID,
		Metric: schema.Metric, Value: value, Unit: schema.Unit, SampledAt: sampledAt, ReceivedAt: receivedAt,
		Quality: QualityAccepted, IdempotencyKey: Identity(deviceID, schema.ID, sampledAt), SourceBatchID: batchID,
	}, nil
}

func Correct(id string, original Observation, value float64, reason string, at time.Time) (Observation, error) {
	if original.CorrectionOf != "" || strings.TrimSpace(reason) == "" || strings.TrimSpace(id) == "" {
		return Observation{}, ErrCorrectionTarget
	}
	corrected := original
	corrected.ID = id
	corrected.Value = value
	corrected.CorrectionOf = original.ID
	corrected.CorrectedAt = at.UTC()
	corrected.ReceivedAt = at.UTC()
	corrected.Quality = QualityAccepted
	corrected.QualityReasons = []string{"manual-correction: " + strings.TrimSpace(reason)}
	corrected.IdempotencyKey = Identity(original.DeviceID, original.SchemaID, original.SampledAt) + ":correction:" + id
	return corrected, nil
}

func (o Observation) EffectiveTime() time.Time {
	return o.SampledAt.UTC()
}
