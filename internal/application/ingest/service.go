package ingest

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/wyw14/cry-082/internal/domain/alert"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/rule"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

var (
	ErrEmptyBatch    = errors.New("empty telemetry batch")
	ErrBatchTooLarge = errors.New("telemetry batch too large")
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type DeviceCatalog interface {
	FindByCode(context.Context, string) (device.Device, error)
	Save(context.Context, device.Device) error
}

type MeasurementCatalog interface {
	FindSchema(context.Context, string) (telemetry.Schema, error)
}

type ObservationJournal interface {
	ExistsIdentity(context.Context, string) (bool, error)
	Latest(context.Context, string, string) (*telemetry.Observation, error)
	Append(context.Context, telemetry.Observation) error
}

type RuleCatalog interface {
	ActiveForSite(context.Context, string, time.Time) ([]rule.Version, error)
}

type EvaluationJournal interface {
	AppendEvaluation(context.Context, rule.Evaluation) error
}

type IncidentSink interface {
	FindMergeable(context.Context, string, alert.Kind, time.Time) (*alert.Alert, error)
	SaveAlert(context.Context, alert.Alert) error
}

type EventBus interface {
	Enqueue(context.Context, string, string, any) error
}

type Dependencies struct {
	Devices      DeviceCatalog
	Schemas      MeasurementCatalog
	Observations ObservationJournal
	Rules        RuleCatalog
	Evaluations  EvaluationJournal
	Alerts       IncidentSink
	Outbox       EventBus
	Transactions transaction.Manager
	Clock        Clock
	IDs          IDGenerator
}

type Sample struct {
	DeviceCode, SchemaID string
	Value                float64
	SampledAt            time.Time
}
type Batch struct {
	BatchID, ActorID string
	ReceivedAt       time.Time
	Samples          []Sample
}
type ItemResult struct {
	Index                         int
	ObservationID, IdempotencyKey string
	Duplicate, Late               bool
	Quality                       telemetry.Quality
	ErrorCode                     string
}
type BatchResult struct {
	BatchID                           string
	Accepted, Duplicates, Quarantined int
	Items                             []ItemResult
}

type Service struct {
	deps     Dependencies
	quality  telemetry.QualityPolicy
	maxBatch int
}

func New(deps Dependencies, quality telemetry.QualityPolicy, maxBatch int) *Service {
	if maxBatch < 1 {
		maxBatch = 500
	}
	return &Service{deps: deps, quality: quality, maxBatch: maxBatch}
}

func (s *Service) Ingest(ctx context.Context, batch Batch) (result BatchResult, err error) {
	if len(batch.Samples) == 0 {
		return BatchResult{}, ErrEmptyBatch
	}
	if len(batch.Samples) > s.maxBatch {
		return BatchResult{}, ErrBatchTooLarge
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result = BatchResult{BatchID: batch.BatchID, Items: make([]ItemResult, 0, len(batch.Samples))}
	tx, err := s.deps.Transactions.Begin(ctx)
	if err != nil {
		return BatchResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	ctx = tx.Bind(ctx)
	indexed := make([]struct {
		index  int
		sample Sample
	}, 0, len(batch.Samples))
	for index, sample := range batch.Samples {
		indexed = append(indexed, struct {
			index  int
			sample Sample
		}{index, sample})
	}
	sort.SliceStable(indexed, func(i, j int) bool { return indexed[i].sample.SampledAt.Before(indexed[j].sample.SampledAt) })
	for _, candidate := range indexed {
		item, itemErr := s.processSample(ctx, batch, candidate.index, candidate.sample)
		if itemErr != nil {
			item.ErrorCode = classify(itemErr)
			result.Items = append(result.Items, item)
			continue
		}
		result.Items = append(result.Items, item)
		if item.Duplicate {
			result.Duplicates++
		} else if item.Quality == telemetry.QualityQuarantined {
			result.Quarantined++
		} else {
			result.Accepted++
		}
	}
	sort.Slice(result.Items, func(i, j int) bool { return result.Items[i].Index < result.Items[j].Index })
	if err = tx.Commit(ctx); err != nil {
		return BatchResult{}, err
	}
	return result, nil
}

func (s *Service) processSample(ctx context.Context, batch Batch, index int, sample Sample) (ItemResult, error) {
	item := ItemResult{Index: index}
	deviceEntity, err := s.deps.Devices.FindByCode(ctx, sample.DeviceCode)
	if err != nil {
		return item, err
	}
	schema, err := s.deps.Schemas.FindSchema(ctx, sample.SchemaID)
	if err != nil {
		return item, err
	}
	identity := telemetry.Identity(deviceEntity.ID, schema.ID, sample.SampledAt)
	item.IdempotencyKey = identity
	exists, err := s.deps.Observations.ExistsIdentity(ctx, identity)
	if err != nil {
		return item, err
	}
	if exists {
		item.Duplicate = true
		return item, nil
	}
	previous, err := s.deps.Observations.Latest(ctx, deviceEntity.ID, schema.ID)
	if err != nil {
		return item, err
	}
	observation, err := telemetry.NewObservation(s.deps.IDs.NewID(), deviceEntity.ID, deviceEntity.SiteID, deviceEntity.PointID, schema, sample.Value, sample.SampledAt, batch.ReceivedAt, batch.BatchID)
	if err != nil {
		return item, err
	}
	quality := s.quality.Evaluate(schema, sample.Value, sample.SampledAt, batch.ReceivedAt, previous)
	observation.Quality, observation.QualityReasons = quality.Quality, quality.Reasons
	item.ObservationID, item.Quality, item.Late = observation.ID, observation.Quality, quality.Late
	if err := s.deps.Observations.Append(ctx, observation); err != nil {
		return item, err
	}
	if err := deviceEntity.MarkSeen(batch.ReceivedAt); err != nil {
		return item, err
	}
	if err := s.deps.Devices.Save(ctx, deviceEntity); err != nil {
		return item, err
	}
	if quality.Late || quality.Quality == telemetry.QualityQuarantined {
		return item, nil
	}
	if err := s.evaluateRules(ctx, observation); err != nil {
		return item, err
	}
	return item, nil
}

func (s *Service) evaluateRules(ctx context.Context, observation telemetry.Observation) error {
	versions, err := s.deps.Rules.ActiveForSite(ctx, observation.SiteID, observation.SampledAt)
	if err != nil {
		return err
	}
	for _, version := range versions {
		evaluation := rule.Evaluate(version, observation.SiteID, observation.PointID, []telemetry.Observation{observation}, "", s.deps.Clock.Now())
		if err := s.deps.Evaluations.AppendEvaluation(ctx, evaluation); err != nil {
			return err
		}
		if !evaluation.Matched {
			continue
		}
		mergeKey := fmt.Sprintf("%s:%s:%s:%d", observation.SiteID, observation.PointID, version.RuleID, version.Version)
		existing, err := s.deps.Alerts.FindMergeable(ctx, mergeKey, alert.KindEnvironmentalExceedance, observation.SampledAt.Add(-version.MergeWindow))
		if err != nil {
			return err
		}
		if existing != nil {
			if err := existing.Merge(observation.SampledAt); err != nil {
				return err
			}
			if err := s.deps.Alerts.SaveAlert(ctx, *existing); err != nil {
				return err
			}
			continue
		}
		created, err := alert.New(s.deps.IDs.NewID(), observation.SiteID, observation.PointID, observation.DeviceID, alert.KindEnvironmentalExceedance, version.RuleID, version.Version, mergeKey, observation.SampledAt)
		if err != nil {
			return err
		}
		if err := s.deps.Alerts.SaveAlert(ctx, created); err != nil {
			return err
		}
		if err := s.deps.Outbox.Enqueue(ctx, "alert.created", created.ID, map[string]any{"alert_id": created.ID, "site_id": created.SiteID, "kind": created.Kind}); err != nil {
			return err
		}
	}
	return nil
}

func classify(err error) string {
	if err == nil {
		return ""
	}
	return "INGEST_ITEM_REJECTED"
}
