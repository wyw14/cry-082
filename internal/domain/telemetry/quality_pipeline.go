package telemetry

import "time"

type QualityStage string

const (
	QualityStageClock QualityStage = "clock"
	QualityStageRange QualityStage = "range"
	QualityStageSpike QualityStage = "spike"
	QualityStageLate  QualityStage = "late"
)

type QualityPipeline struct {
	policy QualityPolicy
	stages []QualityStage
}

func NewQualityPipeline(policy QualityPolicy) QualityPipeline {
	return QualityPipeline{
		policy: policy,
		stages: []QualityStage{
			QualityStageClock,
			QualityStageRange,
			QualityStageSpike,
			QualityStageLate,
		},
	}
}

func (p QualityPipeline) Inspect(schema Schema, value float64, sampledAt, receivedAt time.Time, previous *Observation) QualityResult {
	result := QualityResult{Quality: QualityAccepted}
	for _, stage := range p.stages {
		switch stage {
		case QualityStageClock:
			// A sample whose clock reads ahead of the receiver beyond the
			// configured tolerance is treated as future-sourced and quarantined
			// so it never reaches downstream rule evaluation.
			if sampledAt.After(receivedAt.Add(p.policy.FutureTolerance)) {
				result.Quality = QualityQuarantined
				result.Reasons = append(result.Reasons, "future-sample")
			}
		case QualityStageRange:
			if value < schema.Minimum || value > schema.Maximum {
				result.Quality = QualityQuarantined
				result.Reasons = append(result.Reasons, "out-of-range")
			}
		case QualityStageSpike:
			if previous != nil && p.policy.SpikeMultiplier > 0 {
				delta := value - previous.Value
				if delta < 0 {
					delta = -delta
				}
				if delta > schema.Maximum/p.policy.SpikeMultiplier {
					result.Quality = QualitySuspect
					result.Reasons = append(result.Reasons, "spike")
				}
			}
		case QualityStageLate:
			if p.policy.LateAfter > 0 && receivedAt.Sub(sampledAt) > p.policy.LateAfter {
				result.Late = true
				result.Reasons = append(result.Reasons, "late")
			}
		}
	}
	return result
}
