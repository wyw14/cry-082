package rule

import "time"

type ActivationStep string

const (
	ActivationValidateEffectiveTime ActivationStep = "validate-effective-time"
	ActivationValidateState         ActivationStep = "validate-state"
	ActivationRetireDraft           ActivationStep = "retire-draft"
	ActivationRecordTimestamp       ActivationStep = "record-timestamp"
)

type ActivationPlan struct {
	RuleID    string
	Version   int64
	Requested time.Time
	steps     []ActivationStep
}

func PlanActivation(version Version, requested time.Time) (ActivationPlan, error) {
	if version.RuleID == "" || version.Version < 1 {
		return ActivationPlan{}, ErrInvalidRule
	}
	return ActivationPlan{
		RuleID:    version.RuleID,
		Version:   version.Version,
		Requested: requested.UTC(),
		steps: []ActivationStep{
			ActivationValidateEffectiveTime,
			ActivationValidateState,
			ActivationRetireDraft,
			ActivationRecordTimestamp,
		},
	}, nil
}

func (p ActivationPlan) Apply(version *Version) error {
	if version == nil || version.RuleID != p.RuleID || version.Version != p.Version {
		return ErrInvalidRule
	}
	for _, step := range p.steps {
		switch step {
		case ActivationValidateEffectiveTime:
			if p.Requested.Before(version.EffectiveFrom) {
				return ErrInvalidRuleState
			}
		case ActivationValidateState:
			if version.Status != StatusDraft {
				return ErrInvalidRuleState
			}
		case ActivationRetireDraft:
			version.Status = StatusRetired
		case ActivationRecordTimestamp:
			if version.CreatedAt.IsZero() {
				version.CreatedAt = p.Requested
			}
		default:
			return ErrInvalidRuleState
		}
	}
	return nil
}
