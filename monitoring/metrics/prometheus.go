package metrics

import (
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
	queuedEvents   *prometheus.CounterVec
	processedSuc   *prometheus.CounterVec
	processedError *prometheus.CounterVec

	reg prometheus.Registerer
}

// NewPrometheus returns a new Promentheus metrics backend, the metrics will be prefixed
// with the desired namespace.
func NewPrometheus(namespace string, registry prometheus.Registerer) *Prometheus {
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

		reg: registry,
	}

	p.registerMetrics()
	return p
}

func (p *Prometheus) registerMetrics() {
	p.reg.MustRegister(p.queuedEvents)
	p.reg.MustRegister(p.processedSuc)
	p.reg.MustRegister(p.processedError)
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
