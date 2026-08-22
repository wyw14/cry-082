package httpapi

import "time"

type TelemetrySampleRequest struct {
	DeviceCode string    `json:"device_code" validate:"required,max=64"`
	SchemaID   string    `json:"schema_id" validate:"required,max=64"`
	Value      float64   `json:"value"`
	SampledAt  time.Time `json:"sampled_at" validate:"required"`
}
type TelemetryBatchRequest struct {
	BatchID string                   `json:"batch_id" validate:"required,max=96"`
	Samples []TelemetrySampleRequest `json:"samples" validate:"required,min=1,max=500,dive"`
}
type AlertTransitionRequest struct {
	Status          string `json:"status" validate:"required,oneof=acknowledged dispatched recovering recovered closed open"`
	AssigneeID      string `json:"assignee_id" validate:"omitempty,max=96"`
	Reason          string `json:"reason" validate:"required,min=3,max=500"`
	ExpectedVersion int64  `json:"expected_version" validate:"required,gt=0"`
}
type DeviceTransitionRequest struct {
	Status          string `json:"status" validate:"required,oneof=online offline maintenance replaced retired"`
	ReplacementID   string `json:"replacement_id" validate:"omitempty,max=96"`
	Reason          string `json:"reason" validate:"required,min=3,max=500"`
	ExpectedVersion int64  `json:"expected_version" validate:"required,gt=0"`
}
type CreateRuleRequest struct {
	RuleID             string                 `json:"rule_id" validate:"required,max=64"`
	Name               string                 `json:"name" validate:"required,max=120"`
	Timezone           string                 `json:"timezone" validate:"required"`
	RequireAll         bool                   `json:"require_all"`
	DurationSeconds    int64                  `json:"duration_seconds" validate:"gte=0,lte=86400"`
	MergeWindowSeconds int64                  `json:"merge_window_seconds" validate:"gte=0,lte=86400"`
	LateGraceSeconds   int64                  `json:"late_grace_seconds" validate:"gte=0,lte=86400"`
	EffectiveFrom      time.Time              `json:"effective_from" validate:"required"`
	Conditions         []RuleConditionRequest `json:"conditions" validate:"required,min=1,max=8,dive"`
}
type RuleConditionRequest struct {
	Metric   string  `json:"metric" validate:"required,oneof=pm2_5 pm10 noise temperature humidity wind_speed wind_bearing"`
	Operator string  `json:"operator" validate:"required,oneof=gt gte lt lte"`
	Value    float64 `json:"value"`
}

type ActivateRuleRequest struct {
	Reason string `json:"reason" validate:"required,min=3,max=500"`
}

type RecalculateRuleRequest struct {
	FromVersion int64     `json:"from_version" validate:"required,gt=0,nefield=ToVersion"`
	ToVersion   int64     `json:"to_version" validate:"required,gt=0"`
	WindowStart time.Time `json:"window_start" validate:"required"`
	WindowEnd   time.Time `json:"window_end" validate:"required,gtfield=WindowStart"`
	Reason      string    `json:"reason" validate:"required,min=3,max=500"`
}
