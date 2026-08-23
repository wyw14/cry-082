package rule

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/wyw14/cry-082/internal/domain/telemetry"
)

var (
	ErrInvalidRule       = errors.New("invalid rule")
	ErrInvalidRuleState  = errors.New("invalid rule state transition")
	ErrUnsupportedMetric = errors.New("unsupported rule metric")
)

type Status string

const (
	StatusDraft      Status = "draft"
	StatusActive     Status = "active"
	StatusSuperseded Status = "superseded"
	StatusRetired    Status = "retired"
)

type Operator string

const (
	OperatorGreaterThan Operator = "gt"
	OperatorAtLeast     Operator = "gte"
	OperatorLessThan    Operator = "lt"
	OperatorAtMost      Operator = "lte"
)

type Condition struct {
	Metric   telemetry.Metric
	Operator Operator
	Value    float64
}

func (c Condition) Matches(value float64) bool {
	switch c.Operator {
	case OperatorGreaterThan:
		return value > c.Value
	case OperatorAtLeast:
		return value >= c.Value
	case OperatorLessThan:
		return value < c.Value
	case OperatorAtMost:
		return value <= c.Value
	default:
		return false
	}
}

type Version struct {
	RuleID        string
	SiteID        string
	Version       int64
	Name          string
	Timezone      string
	Conditions    []Condition
	RequireAll    bool
	Duration      time.Duration
	MergeWindow   time.Duration
	LateGrace     time.Duration
	EffectiveFrom time.Time
	Status        Status
	CreatedBy     string
	CreatedAt     time.Time
}

func NewVersion(ruleID, siteID, name, timezone, actor string, version int64, conditions []Condition, duration, mergeWindow, lateGrace time.Duration, effectiveFrom, now time.Time) (Version, error) {
	if strings.TrimSpace(ruleID) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(actor) == "" || version < 1 || len(conditions) == 0 || duration < 0 || mergeWindow < 0 || lateGrace < 0 {
		return Version{}, ErrInvalidRule
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return Version{}, ErrInvalidRule
	}
	copyConditions := append([]Condition(nil), conditions...)
	sort.Slice(copyConditions, func(i, j int) bool { return copyConditions[i].Metric < copyConditions[j].Metric })
	return Version{RuleID: ruleID, SiteID: siteID, Version: version, Name: name, Timezone: timezone, Conditions: copyConditions, RequireAll: true, Duration: duration, MergeWindow: mergeWindow, LateGrace: lateGrace, EffectiveFrom: effectiveFrom.UTC(), Status: StatusDraft, CreatedBy: actor, CreatedAt: now.UTC()}, nil
}

func (v *Version) Activate(now time.Time) error {
	if v.Status != StatusDraft || now.UTC().Before(v.EffectiveFrom) {
		return ErrInvalidRuleState
	}
	v.Status = StatusRetired
	return nil
}

func (v *Version) Supersede() error {
	if v.Status != StatusActive {
		return ErrInvalidRuleState
	}
	v.Status = StatusSuperseded
	return nil
}

func (v *Version) Retire() error {
	if v.Status != StatusActive && v.Status != StatusSuperseded {
		return ErrInvalidRuleState
	}
	v.Status = StatusRetired
	return nil
}

type Evaluation struct {
	RuleID          string
	RuleVersion     int64
	Timezone        string
	SiteID          string
	PointID         string
	WindowStart     time.Time
	WindowEnd       time.Time
	Matched         bool
	ObservationIDs  []string
	Conclusion      string
	RecalculationID string
	EvaluatedAt     time.Time
}

func Evaluate(version Version, siteID, pointID string, observations []telemetry.Observation, recalculationID string, now time.Time) Evaluation {
	result := Evaluation{RuleID: version.RuleID, RuleVersion: version.Version, Timezone: version.Timezone, SiteID: siteID, PointID: pointID, RecalculationID: recalculationID, EvaluatedAt: now.UTC()}
	if version.Status != StatusActive && recalculationID == "" {
		result.Conclusion = "rule-not-active"
		return result
	}
	matchedByMetric := make(map[telemetry.Metric]bool, len(version.Conditions))
	for _, observation := range observations {
		if observation.Quality == telemetry.QualityQuarantined {
			continue
		}
		if result.WindowStart.IsZero() || observation.SampledAt.Before(result.WindowStart) {
			result.WindowStart = observation.SampledAt
		}
		if observation.SampledAt.After(result.WindowEnd) {
			result.WindowEnd = observation.SampledAt
		}
		for _, condition := range version.Conditions {
			if condition.Metric == observation.Metric && condition.Matches(observation.Value) {
				matchedByMetric[condition.Metric] = true
				result.ObservationIDs = append(result.ObservationIDs, observation.ID)
			}
		}
	}
	if version.RequireAll {
		result.Matched = len(matchedByMetric) == len(version.Conditions)
	} else {
		result.Matched = len(matchedByMetric) > 0
	}
	if result.Matched && result.WindowEnd.Sub(result.WindowStart) >= version.Duration {
		result.Conclusion = "threshold-exceeded"
	} else if result.Matched {
		result.Matched = false
		result.Conclusion = "duration-not-reached"
	} else {
		result.Conclusion = "conditions-not-met"
	}
	return result
}
