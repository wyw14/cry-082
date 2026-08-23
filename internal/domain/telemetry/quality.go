package telemetry

import (
	"math"
	"time"
)

type QualityPolicy struct {
	FutureTolerance time.Duration
	LateAfter       time.Duration
	SpikeMultiplier float64
}

type QualityResult struct {
	Quality Quality
	Reasons []string
	Late    bool
}

func (p QualityPolicy) Evaluate(schema Schema, value float64, sampledAt, receivedAt time.Time, previous *Observation) QualityResult {
	result := QualityResult{Quality: QualityAccepted}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < schema.Minimum || value > schema.Maximum {
		result.Quality = QualityQuarantined
		result.Reasons = append(result.Reasons, "outside-schema-range")
	}
	if sampledAt.After(receivedAt.Add(p.FutureTolerance)) {
		result.Quality = QualityQuarantined
		result.Reasons = append(result.Reasons, "future-clock-skew")
	}
	if receivedAt.Sub(sampledAt) > p.LateAfter {
		result.Late = true
		result.Reasons = append(result.Reasons, "late-arrival")
		if result.Quality == QualityAccepted {
			result.Quality = QualitySuspect
		}
	}
	if previous != nil && p.SpikeMultiplier > 1 && previous.Value != 0 {
		ratio := math.Abs(value-previous.Value) / math.Abs(previous.Value)
		if ratio >= p.SpikeMultiplier {
			result.Reasons = append(result.Reasons, "abrupt-spike")
			if result.Quality == QualityAccepted {
				result.Quality = QualitySuspect
			}
		}
	}
	return result
}
