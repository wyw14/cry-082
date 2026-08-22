package telemetry

import (
	"testing"
	"time"
)

func TestObservationIdentityAndCorrectionAreAppendOnly(t *testing.T) {
	sampled := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	schema, err := NewSchema("pm10", MetricPM10, "ug/m3", time.Minute, 0, 2000, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	original, err := NewObservation("o1", "d1", "s1", "p1", schema, 250, sampled, sampled.Add(time.Second), "b1")
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := NewObservation("o2", "d1", "s1", "p1", schema, 250, sampled, sampled.Add(time.Second), "b2")
	if err != nil {
		t.Fatal(err)
	}
	if original.IdempotencyKey != duplicate.IdempotencyKey {
		t.Fatal("identity changed across batches")
	}
	original.Quality = QualityQuarantined
	corrected, err := Correct("o3", original, 125, "reference calibration", sampled.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if corrected.ID == original.ID || corrected.CorrectionOf != original.ID || original.Value != 250 {
		t.Fatalf("append-only correction failed: original=%+v corrected=%+v", original, corrected)
	}
}
