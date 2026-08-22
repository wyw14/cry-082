package ingest

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-082/internal/domain/telemetry"
)

type CorrectionRepository interface {
	FindObservation(context.Context, string) (telemetry.Observation, error)
	Append(context.Context, telemetry.Observation) error
}

type CorrectionService struct {
	observations CorrectionRepository
	clock        Clock
	ids          IDGenerator
}

func NewCorrectionService(observations CorrectionRepository, clock Clock, ids IDGenerator) *CorrectionService {
	return &CorrectionService{observations: observations, clock: clock, ids: ids}
}

func (s *CorrectionService) Correct(ctx context.Context, observationID string, value float64, reason string) (telemetry.Observation, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	original, err := s.observations.FindObservation(ctx, observationID)
	if err != nil {
		return telemetry.Observation{}, err
	}
	if original.Quality != telemetry.QualitySuspect && original.Quality != telemetry.QualityQuarantined {
		return telemetry.Observation{}, errors.New("only suspect or quarantined observations may be corrected")
	}
	corrected, err := telemetry.Correct(s.ids.NewID(), original, value, reason, s.clock.Now())
	if err != nil {
		return telemetry.Observation{}, err
	}
	if err := s.observations.Append(ctx, corrected); err != nil {
		return telemetry.Observation{}, err
	}
	return corrected, nil
}
