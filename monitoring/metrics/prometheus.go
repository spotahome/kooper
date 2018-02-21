package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	promControllerSubsystem = "controller"
)

const (
	addEventType = "add"
	delEventType = "delete"
)

// Prometheus implements the metrics recording in a prometheus registry.
type Prometheus struct {
	// Metrics
	queuedEvents         *prometheus.CounterVec
	processedSuc         *prometheus.CounterVec
	processedError       *prometheus.CounterVec
	processedSucDuration *prometheus.HistogramVec
	processedErrDuration *prometheus.HistogramVec

	reg prometheus.Registerer
}

// NewPrometheus returns a new Prometheus metrics backend with metrics prefixed by the namespace.
func NewPrometheus(namespace string, registry prometheus.Registerer) *Prometheus {
	return NewPrometheusWithBuckets(prometheus.DefBuckets, namespace, registry)
}

// NewPrometheusWithBuckets returns a new Prometheus metrics backend with metrics prefixed by the
// namespace and with custom buckets for the duration/latency metrics. This kind should be used when
// the default buckets don't work. This could happen when the time to process an event is not on the
// range of 5ms-10s duration.
// Check https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-variables
func NewPrometheusWithBuckets(buckets []float64, namespace string, registry prometheus.Registerer) *Prometheus {
	p := &Prometheus{
		queuedEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystem,
			Name:      "queued_events_total",
			Help:      "Total number of events queued.",
		}, []string{"type"}),

		processedSuc: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystem,
			Name:      "processed_events_total",
			Help:      "Total number of successfuly processed events.",
		}, []string{"type"}),

		processedError: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystem,
			Name:      "processed_events_errors_total",
			Help:      "Total number of errors processing events.",
		}, []string{"type"}),

		processedSucDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystem,
			Name:      "processed_events_duration_seconds",
			Help:      "The duration for a successful event to be processed.",
			Buckets:   buckets,
		}, []string{"type"}),

		processedErrDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: promControllerSubsystem,
			Name:      "processed_events_error_duration_seconds",
			Help:      "The duration for a event finished in error to be processed.",
			Buckets:   buckets,
		}, []string{"type"}),

		reg: registry,
	}

	p.registerMetrics()
	return p
}

func (p *Prometheus) registerMetrics() {
	p.reg.MustRegister(p.queuedEvents)
	p.reg.MustRegister(p.processedSuc)
	p.reg.MustRegister(p.processedError)
	p.reg.MustRegister(p.processedSucDuration)
	p.reg.MustRegister(p.processedErrDuration)
}

// IncResourceDeleteEventQueued satisfies metrics.Recorder interface.
func (p *Prometheus) IncResourceDeleteEventQueued() {
	p.queuedEvents.WithLabelValues(delEventType).Inc()
}

// IncResourceAddEventQueued satisfies metrics.Recorder interface.
func (p *Prometheus) IncResourceAddEventQueued() {
	p.queuedEvents.WithLabelValues(addEventType).Inc()
}

// IncResourceAddEventProcessedSuccess satisfies metrics.Recorder interface.
func (p *Prometheus) IncResourceAddEventProcessedSuccess() {
	p.processedSuc.WithLabelValues(addEventType).Inc()
}

// IncResourceAddEventProcessedError satisfies metrics.Recorder interface.
func (p *Prometheus) IncResourceAddEventProcessedError() {
	p.processedError.WithLabelValues(addEventType).Inc()
}

// IncResourceDeleteEventProcessedSuccess satisfies metrics.Recorder interface.
func (p *Prometheus) IncResourceDeleteEventProcessedSuccess() {
	p.processedSuc.WithLabelValues(delEventType).Inc()
}

// IncResourceDeleteEventProcessedError satisfies metrics.Recorder interface.
func (p *Prometheus) IncResourceDeleteEventProcessedError() {
	p.processedError.WithLabelValues(delEventType).Inc()
}

// ObserveDurationResourceAddEventProcessedSuccess satisfies metrics.Recorder interface.
func (p *Prometheus) ObserveDurationResourceAddEventProcessedSuccess(start time.Time) {
	d := p.getDuration(start)
	p.processedSucDuration.WithLabelValues(addEventType).Observe(d.Seconds())
}

// ObserveDurationResourceAddEventProcessedError satisfies metrics.Recorder interface.
func (p *Prometheus) ObserveDurationResourceAddEventProcessedError(start time.Time) {
	d := p.getDuration(start)
	p.processedErrDuration.WithLabelValues(addEventType).Observe(d.Seconds())
}

// ObserveDurationResourceDeleteEventProcessedSuccess satisfies metrics.Recorder interface.
func (p *Prometheus) ObserveDurationResourceDeleteEventProcessedSuccess(start time.Time) {
	d := p.getDuration(start)
	p.processedSucDuration.WithLabelValues(delEventType).Observe(d.Seconds())
}

// ObserveDurationResourceDeleteEventProcessedError satisfies metrics.Recorder interface.
func (p *Prometheus) ObserveDurationResourceDeleteEventProcessedError(start time.Time) {
	d := p.getDuration(start)
	p.processedErrDuration.WithLabelValues(delEventType).Observe(d.Seconds())
}

func (p *Prometheus) getDuration(start time.Time) time.Duration {
	return time.Now().Sub(start)
}
