package controller

import (
	"context"
	"time"
)

// EventType is the event type of a controller enqueued object.
type EventType string

const (
	//AddEvent is the add event.
	AddEvent EventType = "add"
	// DeleteEvent is the delete event.
	DeleteEvent EventType = "delete"
	// RequeueEvent is a requeued event (unknown state when handling again).
	RequeueEvent EventType = "requeue"
)

// MetricsRecorder knows how to record metrics of a controller.
type MetricsRecorder interface {
	// IncResourceEvent increments in one the metric records of a queued event.
	IncResourceEventQueued(ctx context.Context, controller string, eventType EventType)
	// IncResourceEventProcessed increments in one the metric records processed event.
	IncResourceEventProcessed(ctx context.Context, controller string, eventType EventType)
	// IncResourceEventProcessedError increments in one the metric records of a processed event in error.
	IncResourceEventProcessedError(ctx context.Context, controller string, eventType EventType)
	// ObserveDurationResourceEventProcessed measures the duration it took to process a event.
	ObserveDurationResourceEventProcessed(ctx context.Context, controller string, eventType EventType, start time.Time)
}

// DummyMetricsRecorder is a dummy metrics recorder.
var DummyMetricsRecorder = dummy(0)

type dummy int

func (dummy) IncResourceEventQueued(_ context.Context, _ string, _ EventType)                             {}
func (dummy) IncResourceEventProcessed(_ context.Context, _ string, _ EventType)                          {}
func (dummy) IncResourceEventProcessedError(_ context.Context, _ string, _ EventType)                     {}
func (dummy) ObserveDurationResourceEventProcessed(_ context.Context, _ string, _ EventType, _ time.Time) {}
