package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/alert"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

func TestAreaComparisonCountsExceedanceDuration(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	schema, _ := telemetry.NewSchema("pm10", telemetry.MetricPM10, "ug/m3", time.Minute, 0, 2000, time.Minute)
	for index, value := range []float64{100, 180, 190, 120} {
		observation, _ := telemetry.NewObservation(string(rune('a'+index)), "d1", "s1", "p1", schema, value, now.Add(time.Duration(index)*time.Minute), now.Add(time.Duration(index)*time.Minute), "b1")
		if err := store.Append(ctx, observation); err != nil {
			t.Fatal(err)
		}
	}
	offline, _ := alert.New("a1", "s1", "p1", "d1", alert.KindDeviceOffline, "", 0, "d1:offline", now)
	if err := store.SaveAlert(ctx, offline); err != nil {
		t.Fatal(err)
	}
	service := New(store, store)
	comparison, err := service.CompareAreas(ctx, "s1", telemetry.MetricPM10, 150, now, now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(comparison) != 1 || comparison[0].ExceedanceSeconds != 120 {
		t.Fatalf("comparison=%+v", comparison)
	}
	dashboard, err := service.Dashboard(ctx, "s1", now.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.OpenOfflineAlerts != 1 || dashboard.OpenEnvironmentalAlerts != 0 {
		t.Fatalf("dashboard=%+v", dashboard)
	}
}
