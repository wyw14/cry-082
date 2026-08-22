package rule

import (
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/telemetry"
)

func TestEvaluationBindsVersionAndTimezone(t *testing.T) {
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	version, err := NewVersion("r1", "s1", "PM10持续超标", "Asia/Shanghai", "u1", 3, []Condition{{Metric: telemetry.MetricPM10, Operator: OperatorAtLeast, Value: 150}}, 5*time.Minute, 10*time.Minute, time.Minute, now.Add(-time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := version.Activate(now); err != nil {
		t.Fatal(err)
	}
	schema, _ := telemetry.NewSchema("pm10", telemetry.MetricPM10, "ug/m3", time.Minute, 0, 2000, time.Minute)
	first, _ := telemetry.NewObservation("o1", "d1", "s1", "p1", schema, 180, now, now, "b1")
	last, _ := telemetry.NewObservation("o2", "d1", "s1", "p1", schema, 190, now.Add(6*time.Minute), now.Add(6*time.Minute), "b1")
	evaluation := Evaluate(version, "s1", "p1", []telemetry.Observation{first, last}, "", now.Add(7*time.Minute))
	if !evaluation.Matched || evaluation.RuleVersion != 3 || evaluation.Timezone != "Asia/Shanghai" {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}
