package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spotahome/kooper/controller"
)

const (
	promNamespace           = "kooper"
	promControllerSubsystem = "controller"
)

// Config is the Recorder Config.
type Config struct {
	// Registerer is a prometheus registerer, e.g: prometheus.Registry.
	// By default will use Prometheus default registry.
	Registerer prometheus.Registerer
	// Buckets sets custom buckets for the duration/latency metrics. This should be used when
	// the default buckets don't work. This could happen when the time to process an event is not on the
	// range of 5ms-10s duration.
	// Check https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-variables
	Buckets []float64
}

func (c *Config) defaults() {
	if c.Registerer == nil {
		c.Registerer = prometheus.DefaultRegisterer
	}

	if c.Buckets == nil || len(c.Buckets) == 0 {
		c.Buckets = prometheus.DefBuckets
	}
}

// Recorder implements the metrics recording in a prometheus registry.
type Recorder struct {
	// Metrics
	queuedEvents           *prometheus.CounterVec
	processedEvents        *prometheus.CounterVec
	processedEventErrors   *prometheus.CounterVec
	processedEventDuration *prometheus.HistogramVec
}

// New returns a new Prometheus implementaiton for a metrics recorder.
func New(cfg Config) *Recorder {
	cfg.defaults()

	r := &Recorder{
		queuedEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: promNamespace,
			Subsystem: promControllerSubsystem,
			Name:      "queued_events_total",
			Help:      "Total number of events queued.",
		}, []string{"controller", "type"}),

		processedEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: promNamespace,
			Subsystem: promControllerSubsystem,
			Name:      "processed_events_total",
			Help:      "Total number of successfuly processed events.",
		}, []string{"controller", "type"}),

		processedEventErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: promNamespace,
			Subsystem: promControllerSubsystem,
			Name:      "processed_event_errors_total",
			Help:      "Total number of errors processing events.",
		}, []string{"controller", "type"}),

		processedEventDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: promNamespace,
			Subsystem: promControllerSubsystem,
			Name:      "processed_event_duration_seconds",
			Help:      "The duration for a successful event to be processed.",
			Buckets:   cfg.Buckets,
		}, []string{"controller", "type"}),
	}

	// Register metrics.
	cfg.Registerer.MustRegister(
		r.queuedEvents,
		r.processedEvents,
		r.processedEventErrors,
		r.processedEventDuration)

	return r
}

// IncResourceEventQueued satisfies metrics.Recorder interface.
func (r Recorder) IncResourceEventQueued(_ context.Context, controller string, eventType controller.EventType) {
	r.queuedEvents.WithLabelValues(controller, string(eventType)).Inc()
}

// IncResourceEventProcessed satisfies metrics.Recorder interface.
func (r Recorder) IncResourceEventProcessed(_ context.Context, controller string, eventType controller.EventType) {
	r.processedEvents.WithLabelValues(controller, string(eventType)).Inc()
}

// IncResourceEventProcessedError satisfies metrics.Recorder interface.
func (r Recorder) IncResourceEventProcessedError(_ context.Context, controller string, eventType controller.EventType) {
	r.processedEventErrors.WithLabelValues(controller, string(eventType)).Inc()
}

// ObserveDurationResourceEventProcessed satisfies metrics.Recorder interface.
func (r Recorder) ObserveDurationResourceEventProcessed(_ context.Context, controller string, eventType controller.EventType, start time.Time) {
	secs := time.Now().Sub(start).Seconds()
	r.processedEventDuration.WithLabelValues(controller, string(eventType)).Observe(secs)
}

// Check we implement all the required metrics recorder interfaces.
var _ controller.MetricsRecorder = &Recorder{}
