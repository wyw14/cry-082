package telemetry

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidSchema = errors.New("invalid measurement schema")

type Metric string

const (
	MetricPM25        Metric = "pm2_5"
	MetricPM10        Metric = "pm10"
	MetricNoise       Metric = "noise"
	MetricTemperature Metric = "temperature"
	MetricHumidity    Metric = "humidity"
	MetricWindSpeed   Metric = "wind_speed"
	MetricWindBearing Metric = "wind_bearing"
)

type Schema struct {
	ID             string
	Metric         Metric
	Unit           string
	SamplingPeriod time.Duration
	Minimum        float64
	Maximum        float64
	MaxClockSkew   time.Duration
	Version        int64
}

func NewSchema(id string, metric Metric, unit string, period time.Duration, min, max float64, skew time.Duration) (Schema, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(unit) == "" || period <= 0 || max <= min || skew < 0 {
		return Schema{}, ErrInvalidSchema
	}
	supported := map[Metric]bool{MetricPM25: true, MetricPM10: true, MetricNoise: true, MetricTemperature: true, MetricHumidity: true, MetricWindSpeed: true, MetricWindBearing: true}
	if !supported[metric] {
		return Schema{}, ErrInvalidSchema
	}
	return Schema{ID: id, Metric: metric, Unit: unit, SamplingPeriod: period, Minimum: min, Maximum: max, MaxClockSkew: skew, Version: 1}, nil
}
