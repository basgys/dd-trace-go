// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package gqlgen

import (
	"math"

	"github.com/basgys/dd-trace-go/internal/globalconfig"
	"github.com/basgys/dd-trace-go/internal/namingschema"
)

const defaultServiceName = "graphql"

type config struct {
	serviceName   string
	analyticsRate float64
	tags          map[string]interface{}
}

// An Option configures the gqlgen integration.
type Option func(cfg *config)

func defaults(cfg *config) {
	cfg.serviceName = namingschema.ServiceNameOverrideV0(defaultServiceName, defaultServiceName)
	cfg.analyticsRate = globalconfig.AnalyticsRate()
	cfg.tags = make(map[string]interface{})
}

// WithAnalytics enables or disables Trace Analytics for all started spans.
func WithAnalytics(on bool) Option {
	if on {
		return WithAnalyticsRate(1.0)
	}
	return WithAnalyticsRate(math.NaN())
}

// WithAnalyticsRate sets the sampling rate for Trace Analytics events correlated to started spans.
func WithAnalyticsRate(rate float64) Option {
	return func(cfg *config) {
		cfg.analyticsRate = rate
	}
}

// WithServiceName sets the given service name for the gqlgen server.
func WithServiceName(name string) Option {
	return func(cfg *config) {
		cfg.serviceName = name
	}
}

// WithCustomTag will attach the value to the span tagged by the key.
func WithCustomTag(key string, value interface{}) Option {
	return func(cfg *config) {
		if cfg.tags == nil {
			cfg.tags = make(map[string]interface{})
		}
		cfg.tags[key] = value
	}
}
