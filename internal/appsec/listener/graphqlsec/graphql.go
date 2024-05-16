// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package graphqlsec

import (
	"sync"

	"github.com/DataDog/appsec-internal-go/limiter"
	waf "github.com/DataDog/go-libddwaf/v2"
	"github.com/basgys/dd-trace-go/ddtrace/ext"
	"github.com/basgys/dd-trace-go/internal/appsec/config"
	"github.com/basgys/dd-trace-go/internal/appsec/dyngo"
	"github.com/basgys/dd-trace-go/internal/appsec/emitter/graphqlsec/types"
	"github.com/basgys/dd-trace-go/internal/appsec/emitter/sharedsec"
	"github.com/basgys/dd-trace-go/internal/appsec/listener"
	shared "github.com/basgys/dd-trace-go/internal/appsec/listener/sharedsec"
	"github.com/basgys/dd-trace-go/internal/appsec/trace"
	"github.com/basgys/dd-trace-go/internal/log"
	"github.com/basgys/dd-trace-go/internal/samplernames"
)

// GraphQL rule addresses currently supported by the WAF
const (
	graphQLServerResolverAddr = "graphql.server.resolver"
)

// List of GraphQL rule addresses currently supported by the WAF
var supportedAddresses = listener.AddressSet{
	graphQLServerResolverAddr: {},
}

// Install registers the GraphQL WAF Event Listener on the given root operation.
func Install(wafHandle *waf.Handle, _ sharedsec.Actions, cfg *config.Config, lim limiter.Limiter, root dyngo.Operation) {
	if listener := newWafEventListener(wafHandle, cfg, lim); listener != nil {
		log.Debug("appsec: registering the GraphQL WAF Event Listener")
		dyngo.On(root, listener.onEvent)
	}
}

type wafEventListener struct {
	wafHandle *waf.Handle
	config    *config.Config
	addresses map[string]struct{}
	limiter   limiter.Limiter
	wafDiags  waf.Diagnostics
	once      sync.Once
}

func newWafEventListener(wafHandle *waf.Handle, cfg *config.Config, limiter limiter.Limiter) *wafEventListener {
	if wafHandle == nil {
		log.Debug("appsec: no WAF Handle available, the GraphQL WAF Event Listener will not be registered")
		return nil
	}

	addresses := listener.FilterAddressSet(supportedAddresses, wafHandle)
	if len(addresses) == 0 {
		log.Debug("appsec: no supported GraphQL address is used by currently loaded WAF rules, the GraphQL WAF Event Listener will not be registered")
		return nil
	}

	return &wafEventListener{
		wafHandle: wafHandle,
		config:    cfg,
		addresses: addresses,
		limiter:   limiter,
		wafDiags:  wafHandle.Diagnostics(),
	}
}

// NewWAFEventListener returns the WAF event listener to register in order
// to enable it.
func (l *wafEventListener) onEvent(request *types.RequestOperation, _ types.RequestOperationArgs) {
	wafCtx := waf.NewContextWithBudget(l.wafHandle, l.config.WAFTimeout)
	if wafCtx == nil {
		return
	}

	// Add span tags notifying this trace is AppSec-enabled
	trace.SetAppSecEnabledTags(request)
	l.once.Do(func() {
		shared.AddRulesMonitoringTags(request, &l.wafDiags)
		request.SetTag(ext.ManualKeep, samplernames.AppSec)
	})

	dyngo.On(request, func(query *types.ExecutionOperation, args types.ExecutionOperationArgs) {
		dyngo.On(query, func(field *types.ResolveOperation, args types.ResolveOperationArgs) {
			if _, found := l.addresses[graphQLServerResolverAddr]; found {
				wafResult := shared.RunWAF(
					wafCtx,
					waf.RunAddressData{
						Ephemeral: map[string]any{
							graphQLServerResolverAddr: map[string]any{args.FieldName: args.Arguments},
						},
					},
					l.config.WAFTimeout,
				)
				shared.AddSecurityEvents(field, l.limiter, wafResult.Events)
			}

			dyngo.OnFinish(field, func(field *types.ResolveOperation, res types.ResolveOperationRes) {
				trace.SetEventSpanTags(field, field.Events())
			})
		})

		dyngo.OnFinish(query, func(query *types.ExecutionOperation, res types.ExecutionOperationRes) {
			trace.SetEventSpanTags(query, query.Events())
		})
	})

	dyngo.OnFinish(request, func(request *types.RequestOperation, res types.RequestOperationRes) {
		defer wafCtx.Close()

		shared.AddWAFMonitoringTags(request, l.wafDiags.Version, wafCtx.Stats().Metrics())
		trace.SetEventSpanTags(request, request.Events())
	})
}
