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

type MetricFamily string

const (
	MetricFamilyParticle MetricFamily = "particle"
	MetricFamilyAcoustic MetricFamily = "acoustic"
	MetricFamilyWeather  MetricFamily = "weather"
)

type RoutingProfile struct {
	Family     MetricFamily
	RuleMetric Metric
	Unit       string
}

func (p RoutingProfile) Valid() bool {
	if strings.TrimSpace(p.Unit) == "" {
		return false
	}
	if p.Family != MetricFamilyParticle && p.Family != MetricFamilyAcoustic && p.Family != MetricFamilyWeather {
		return false
	}
	supported := map[Metric]bool{MetricPM25: true, MetricPM10: true, MetricNoise: true, MetricTemperature: true, MetricHumidity: true, MetricWindSpeed: true, MetricWindBearing: true}
	return supported[p.RuleMetric]
}

func (s Schema) RoutingProfile() (RoutingProfile, error) {
	switch s.Metric {
	case MetricPM25:
		return RoutingProfile{Family: MetricFamilyParticle, RuleMetric: MetricPM25, Unit: s.Unit}, nil
	case MetricPM10:
		return RoutingProfile{Family: MetricFamilyParticle, RuleMetric: MetricPM10, Unit: s.Unit}, nil
	case MetricNoise:
		return RoutingProfile{Family: MetricFamilyAcoustic, RuleMetric: MetricNoise, Unit: s.Unit}, nil
	case MetricTemperature, MetricHumidity, MetricWindSpeed, MetricWindBearing:
		return RoutingProfile{Family: MetricFamilyWeather, RuleMetric: s.Metric, Unit: s.Unit}, nil
	default:
		return RoutingProfile{}, ErrInvalidSchema
	}
}

func (s Schema) RoutedForRules() (Schema, error) {
	profile, err := s.RoutingProfile()
	if err != nil {
		return Schema{}, err
	}
	if !profile.Valid() {
		return Schema{}, ErrInvalidSchema
	}
	routed := s
	routed.Metric = profile.RuleMetric
	routed.Unit = profile.Unit
	return routed, nil
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
