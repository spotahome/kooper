package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/spotahome/kooper/monitoring/metrics"
)

// Handler knows how to handle the received resources from a kubernetes cluster.
type Handler interface {
	Add(context.Context, runtime.Object) error
	Delete(context.Context, string) error
}

// AddFunc knows how to handle resource adds.
type AddFunc func(context.Context, runtime.Object) error

// DeleteFunc knows how to handle resource deletes.
type DeleteFunc func(context.Context, string) error

// HandlerFunc is a handler that is created from functions that the
// Handler interface requires.
type HandlerFunc struct {
	AddFunc    AddFunc
	DeleteFunc DeleteFunc
}

// Add satisfies Handler interface.
func (h *HandlerFunc) Add(ctx context.Context, obj runtime.Object) error {
	if h.AddFunc == nil {
		return fmt.Errorf("function can't be nil")
	}
	return h.AddFunc(ctx, obj)
}

// Delete satisfies Handler interface.
func (h *HandlerFunc) Delete(ctx context.Context, s string) error {
	if h.DeleteFunc == nil {
		return fmt.Errorf("function can't be nil")
	}
	return h.DeleteFunc(ctx, s)
}

type metricsMeasuredHandler struct {
	id   string
	mrec metrics.Recorder
	next Handler
}

func newMetricsMeasuredHandler(id string, mrec metrics.Recorder, next Handler) Handler {
	return metricsMeasuredHandler{
		id:   id,
		mrec: mrec,
		next: next,
	}
}

func (m metricsMeasuredHandler) Add(ctx context.Context, obj runtime.Object) (err error) {
	defer func(start time.Time) {
		m.mrec.ObserveDurationResourceEventProcessed(m.id, metrics.AddEvent, start)

		if err != nil {
			m.mrec.IncResourceEventProcessedError(m.id, metrics.AddEvent)
		}
	}(time.Now())

	m.mrec.IncResourceEventProcessed(m.id, metrics.AddEvent)

	return m.next.Add(ctx, obj)
}

func (m metricsMeasuredHandler) Delete(ctx context.Context, objKey string) (err error) {
	defer func(start time.Time) {
		m.mrec.ObserveDurationResourceEventProcessed(m.id, metrics.DeleteEvent, start)

		if err != nil {
			m.mrec.IncResourceEventProcessedError(m.id, metrics.DeleteEvent)
		}
	}(time.Now())

	m.mrec.IncResourceEventProcessed(m.id, metrics.DeleteEvent)

	return m.next.Delete(ctx, objKey)
}
