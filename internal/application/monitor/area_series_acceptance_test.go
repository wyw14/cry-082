package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/telemetry"
)

type areaObservationRepo struct{ values []telemetry.Observation }

func (r areaObservationRepo) Range(context.Context, string, time.Time, time.Time) ([]telemetry.Observation, error) {
	return append([]telemetry.Observation(nil), r.values...), nil
}

func TestAreaComparisonKeepsMonitoringPointsIndependent(t *testing.T) {
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	repo := areaObservationRepo{values: []telemetry.Observation{
		{ID: "north-1", SiteID: "site-1", PointID: "north", Metric: telemetry.MetricPM10, Value: 20, SampledAt: now, Quality: telemetry.QualityAccepted},
		{ID: "north-2", SiteID: "site-1", PointID: "north", Metric: telemetry.MetricPM10, Value: 40, SampledAt: now.Add(time.Minute), Quality: telemetry.QualityAccepted},
		{ID: "south-1", SiteID: "site-1", PointID: "south", Metric: telemetry.MetricPM10, Value: 180, SampledAt: now, Quality: telemetry.QualityAccepted},
		{ID: "south-2", SiteID: "site-1", PointID: "south", Metric: telemetry.MetricPM10, Value: 220, SampledAt: now.Add(time.Minute), Quality: telemetry.QualityAccepted},
	}}
	result, err := New(repo, nil).CompareAreas(context.Background(), "site-1", telemetry.MetricPM10, 150, now.Add(-time.Minute), now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0].PointID != "north" || result[0].Average != 30 || result[0].Samples != 2 || result[1].PointID != "south" || result[1].Average != 200 || result[1].Samples != 2 {
		t.Fatalf("monitoring point series were mixed: %+v", result)
	}
}
