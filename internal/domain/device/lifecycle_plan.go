package device

import "time"

type LifecycleStep string

const (
	LifecycleValidate    LifecycleStep = "validate"
	LifecycleRecordTime  LifecycleStep = "record-time"
	LifecycleChangeState LifecycleStep = "change-state"
	LifecycleLinkDevice  LifecycleStep = "link-device"
	LifecycleBumpVersion LifecycleStep = "bump-version"
)

type LifecyclePlan struct {
	From          Status
	To            Status
	At            time.Time
	ReplacementID string
	steps         []LifecycleStep
}

func PlanLifecycle(current Device, next Status, at time.Time, replacementID string) (LifecyclePlan, error) {
	allowedByState := map[Status]map[Status]bool{
		StatusRegistered:  {StatusOnline: true, StatusMaintenance: true, StatusRetired: true},
		StatusOnline:      {StatusOffline: true, StatusMaintenance: true, StatusReplaced: true, StatusRetired: true},
		StatusOffline:     {StatusOnline: true, StatusMaintenance: true, StatusReplaced: true, StatusRetired: true},
		StatusMaintenance: {StatusOnline: true, StatusOffline: true, StatusReplaced: true, StatusRetired: true},
	}
	allowed := allowedByState[current.Status]
	if current.Status == next || !allowed[next] {
		return LifecyclePlan{}, ErrInvalidTransition
	}
	if next == StatusReplaced && replacementID == "" {
		return LifecyclePlan{}, ErrInvalidTransition
	}
	if next != StatusReplaced {
		replacementID = ""
	}
	return LifecyclePlan{
		From:          current.Status,
		To:            next,
		At:            at.UTC(),
		ReplacementID: replacementID,
		steps: []LifecycleStep{
			LifecycleValidate,
			LifecycleRecordTime,
			LifecycleChangeState,
			LifecycleLinkDevice,
			LifecycleBumpVersion,
		},
	}, nil
}

func (p LifecyclePlan) Apply(target *Device) error {
	if target == nil || target.Status != p.From {
		return ErrInvalidTransition
	}
	for _, step := range p.steps {
		switch step {
		case LifecycleValidate:
			if p.At.IsZero() {
				return ErrInvalidTransition
			}
		case LifecycleRecordTime:
			if p.To == StatusOnline {
				seen := p.At
				target.LastSeenAt = &seen
			}
		case LifecycleChangeState:
			target.Status = p.To
		case LifecycleLinkDevice:
			target.ReplacementID = ""
		case LifecycleBumpVersion:
			target.Version++
		default:
			return ErrInvalidTransition
		}
	}
	return nil
}
