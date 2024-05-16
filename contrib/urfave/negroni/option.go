// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Package negroni provides helper functions for tracing the urfave/negroni package (https://github.com/urfave/negroni).
package negroni

import (
	"math"
	"net/http"

	"github.com/basgys/dd-trace-go/ddtrace"
	"github.com/basgys/dd-trace-go/internal"
	"github.com/basgys/dd-trace-go/internal/globalconfig"
	"github.com/basgys/dd-trace-go/internal/namingschema"
	"github.com/basgys/dd-trace-go/internal/normalizer"
)

const defaultServiceName = "negroni.router"

type config struct {
	serviceName   string
	spanOpts      []ddtrace.StartSpanOption // additional span options to be applied
	analyticsRate float64
	isStatusError func(statusCode int) bool
	resourceNamer func(r *http.Request) string
	headerTags    *internal.LockMap
}

// Option represents an option that can be passed to NewRouter.
type Option func(*config)

func defaults(cfg *config) {
	cfg.serviceName = namingschema.ServiceName(defaultServiceName)
	if internal.BoolEnv("DD_TRACE_NEGRONI_ANALYTICS_ENABLED", false) {
		cfg.analyticsRate = 1.0
	} else {
		cfg.analyticsRate = globalconfig.AnalyticsRate()
	}
	cfg.headerTags = globalconfig.HeaderTagMap()
	cfg.isStatusError = isServerError
	cfg.resourceNamer = defaultResourceNamer
}

// WithServiceName sets the given service name for the router.
func WithServiceName(name string) Option {
	return func(cfg *config) {
		cfg.serviceName = name
	}
}

// WithSpanOptions applies the given set of options to the spans started
// by the router.
func WithSpanOptions(opts ...ddtrace.StartSpanOption) Option {
	return func(cfg *config) {
		cfg.spanOpts = opts
	}
}

// WithAnalytics enables Trace Analytics for all started spans.
func WithAnalytics(on bool) Option {
	return func(cfg *config) {
		if on {
			cfg.analyticsRate = 1.0
		} else {
			cfg.analyticsRate = math.NaN()
		}
	}
}

// WithAnalyticsRate sets the sampling rate for Trace Analytics events
// correlated to started spans.
func WithAnalyticsRate(rate float64) Option {
	return func(cfg *config) {
		if rate >= 0.0 && rate <= 1.0 {
			cfg.analyticsRate = rate
		} else {
			cfg.analyticsRate = math.NaN()
		}
	}
}

// WithStatusCheck specifies a function fn which reports whether the passed
// statusCode should be considered an error.
func WithStatusCheck(fn func(statusCode int) bool) Option {
	return func(cfg *config) {
		cfg.isStatusError = fn
	}
}

func isServerError(statusCode int) bool {
	return statusCode >= 500 && statusCode < 600
}

// WithResourceNamer specifies a function which will be used to obtain a resource name for a given
// negroni request, using the request's context.
func WithResourceNamer(namer func(r *http.Request) string) Option {
	return func(cfg *config) {
		cfg.resourceNamer = namer
	}
}

func defaultResourceNamer(_ *http.Request) string {
	return ""
}

// WithHeaderTags enables the integration to attach HTTP request headers as span tags.
// Warning:
// Using this feature can risk exposing sensitive data such as authorization tokens to Datadog.
// Special headers can not be sub-selected. E.g., an entire Cookie header would be transmitted, without the ability to choose specific Cookies.
func WithHeaderTags(headers []string) Option {
	headerTagsMap := normalizer.HeaderTagSlice(headers)
	return func(cfg *config) {
		cfg.headerTags = internal.NewLockMap(headerTagsMap)
	}
}
