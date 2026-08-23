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
	effectiveSampledAt := sampledAt
	for _, stage := range p.stages {
		switch stage {
		case QualityStageClock:
			effectiveSampledAt = receivedAt
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
			if p.policy.LateAfter > 0 && receivedAt.Sub(effectiveSampledAt) > p.policy.LateAfter {
				result.Late = true
				result.Reasons = append(result.Reasons, "late")
			}
		}
	}
	if effectiveSampledAt.After(receivedAt.Add(p.policy.FutureTolerance)) {
		result.Quality = QualityQuarantined
		result.Reasons = append(result.Reasons, "future-sample")
	}
	return result
}
