package monitor

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/wyw14/cry-082/internal/domain/alert"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
)

var ErrInvalidWindow = errors.New("invalid monitoring window")

type ObservationRepository interface {
	Range(context.Context, string, time.Time, time.Time) ([]telemetry.Observation, error)
}
type AlertRepository interface {
	RangeAlerts(context.Context, string, time.Time, time.Time) ([]alert.Alert, error)
}

type Service struct {
	observations ObservationRepository
	alerts       AlertRepository
}

func New(observations ObservationRepository, alerts AlertRepository) *Service {
	return &Service{observations: observations, alerts: alerts}
}

type LatestMetric struct {
	PointID   string
	Metric    telemetry.Metric
	Value     float64
	Unit      string
	SampledAt time.Time
	Quality   telemetry.Quality
}
type Dashboard struct {
	SiteID                                     string
	GeneratedAt                                time.Time
	Latest                                     []LatestMetric
	OpenEnvironmentalAlerts, OpenOfflineAlerts int
}

func (s *Service) Dashboard(ctx context.Context, siteID string, now time.Time) (Dashboard, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	observations, err := s.observations.Range(ctx, siteID, now.Add(-24*time.Hour), now)
	if err != nil {
		return Dashboard{}, err
	}
	alerts, err := s.alerts.RangeAlerts(ctx, siteID, now.Add(-30*24*time.Hour), now)
	if err != nil {
		return Dashboard{}, err
	}
	latest := make(map[string]telemetry.Observation)
	for _, observation := range observations {
		key := observation.PointID + ":" + string(observation.Metric)
		current, ok := latest[key]
		if !ok || observation.SampledAt.After(current.SampledAt) {
			latest[key] = observation
		}
	}
	dashboard := Dashboard{SiteID: siteID, GeneratedAt: now.UTC()}
	for _, observation := range latest {
		dashboard.Latest = append(dashboard.Latest, LatestMetric{PointID: observation.PointID, Metric: observation.Metric, Value: observation.Value, Unit: observation.Unit, SampledAt: observation.SampledAt, Quality: observation.Quality})
	}
	sort.Slice(dashboard.Latest, func(i, j int) bool {
		if dashboard.Latest[i].PointID == dashboard.Latest[j].PointID {
			return dashboard.Latest[i].Metric < dashboard.Latest[j].Metric
		}
		return dashboard.Latest[i].PointID < dashboard.Latest[j].PointID
	})
	for _, item := range alerts {
		if item.Status == alert.StatusClosed {
			continue
		}
		if item.Kind == alert.KindDeviceOffline {
			dashboard.OpenOfflineAlerts++
		} else if item.Kind == alert.KindEnvironmentalExceedance {
			dashboard.OpenEnvironmentalAlerts++
		}
	}
	return dashboard, nil
}

type TrendPoint struct {
	Bucket                    time.Time
	Minimum, Maximum, Average float64
	Samples                   int
}

func (s *Service) Trend(ctx context.Context, siteID, pointID string, metric telemetry.Metric, start, end time.Time, bucket time.Duration) ([]TrendPoint, error) {
	if !start.Before(end) || bucket <= 0 || end.Sub(start) > 90*24*time.Hour {
		return nil, ErrInvalidWindow
	}
	observations, err := s.observations.Range(ctx, siteID, start, end)
	if err != nil {
		return nil, err
	}
	type aggregate struct {
		min, max, sum float64
		count         int
	}
	aggregates := map[time.Time]aggregate{}
	for _, observation := range observations {
		if observation.PointID != pointID || observation.Metric != metric || observation.Quality == telemetry.QualityQuarantined {
			continue
		}
		key := observation.SampledAt.Truncate(bucket)
		current := aggregates[key]
		if current.count == 0 || observation.Value < current.min {
			current.min = observation.Value
		}
		if current.count == 0 || observation.Value > current.max {
			current.max = observation.Value
		}
		current.sum += observation.Value
		current.count++
		aggregates[key] = current
	}
	result := make([]TrendPoint, 0, len(aggregates))
	for bucketStart, aggregate := range aggregates {
		result = append(result, TrendPoint{Bucket: bucketStart, Minimum: aggregate.min, Maximum: aggregate.max, Average: aggregate.sum / float64(aggregate.count), Samples: aggregate.count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Bucket.Before(result[j].Bucket) })
	return result, nil
}

type AreaComparison struct {
	PointID           string
	Average, Maximum  float64
	ExceedanceSeconds int64
	Samples           int
}

func (s *Service) CompareAreas(ctx context.Context, siteID string, metric telemetry.Metric, threshold float64, start, end time.Time) ([]AreaComparison, error) {
	if !start.Before(end) {
		return nil, ErrInvalidWindow
	}
	observations, err := s.observations.Range(ctx, siteID, start, end)
	if err != nil {
		return nil, err
	}
	type aggregate struct {
		sum, max     float64
		samples      int
		last         time.Time
		lastExceeded bool
		seconds      int64
	}
	byPoint := map[string]aggregate{}
	sort.Slice(observations, func(i, j int) bool { return observations[i].SampledAt.Before(observations[j].SampledAt) })
	for _, observation := range observations {
		if observation.Metric != metric || observation.Quality == telemetry.QualityQuarantined {
			continue
		}
		a := byPoint[observation.PointID]
		if a.samples > 0 && a.lastExceeded {
			a.seconds += int64(observation.SampledAt.Sub(a.last).Seconds())
		}
		a.sum += observation.Value
		if a.samples == 0 || observation.Value > a.max {
			a.max = observation.Value
		}
		a.samples++
		a.last = observation.SampledAt
		a.lastExceeded = observation.Value >= threshold
		byPoint[observation.PointID] = a
	}
	result := make([]AreaComparison, 0, len(byPoint))
	for pointID, a := range byPoint {
		result = append(result, AreaComparison{PointID: pointID, Average: a.sum / float64(a.samples), Maximum: a.max, ExceedanceSeconds: a.seconds, Samples: a.samples})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PointID < result[j].PointID })
	return result, nil
}
