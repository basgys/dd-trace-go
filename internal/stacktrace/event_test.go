// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package stacktrace

import (
	"testing"

	"github.com/basgys/dd-trace-go/ddtrace/mocktracer"
	ddtracer "github.com/basgys/dd-trace-go/ddtrace/tracer"
	"github.com/basgys/dd-trace-go/internal"

	"github.com/stretchr/testify/require"
)

func TestNewEvent(t *testing.T) {
	event := NewEvent(ExceptionEvent, "", "message")
	require.Equal(t, ExceptionEvent, event.Category)
	require.Equal(t, "go", event.Language)
	require.Equal(t, "message", event.Message)
	require.GreaterOrEqual(t, len(event.Frames), 3)
	require.Equal(t, "TestNewEvent", event.Frames[0].Function)
}

func TestEventToSpan(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	span := ddtracer.StartSpan("op")
	event := NewEvent(ExceptionEvent, "", "message")
	AddToSpan(span, event)
	span.Finish()

	spans := mt.FinishedSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "op", spans[0].OperationName())

	eventsMap := spans[0].Tag("_dd.stack").(internal.MetaStructValue).Value.(map[EventCategory][]*Event)
	require.Len(t, eventsMap, 3)

	eventsCat := eventsMap[ExceptionEvent]
	require.Len(t, eventsCat, 1)

	require.Equal(t, *event, *eventsCat[0])
}
