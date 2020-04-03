package prometheus

import (
	"context"
	"strconv"
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
	// InQueueBuckets sets custom buckets for the duration/latency items in queue metrics.
	// Check https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-variables
	InQueueBuckets []float64
	// ProcessingBuckets sets custom buckets for the duration/latency processing metrics.
	// Check https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-variables
	ProcessingBuckets []float64
}

func (c *Config) defaults() {
	if c.Registerer == nil {
		c.Registerer = prometheus.DefaultRegisterer
	}

	if c.InQueueBuckets == nil || len(c.InQueueBuckets) == 0 {
		c.InQueueBuckets = prometheus.DefBuckets
	}

	if c.ProcessingBuckets == nil || len(c.ProcessingBuckets) == 0 {
		c.ProcessingBuckets = prometheus.DefBuckets
	}
}

// Recorder implements the metrics recording in a prometheus registry.
type Recorder struct {
	queuedEventsTotal      *prometheus.CounterVec
	inQueueEventDuration   *prometheus.HistogramVec
	processedEventDuration *prometheus.HistogramVec
}

// New returns a new Prometheus implementaiton for a metrics recorder.
func New(cfg Config) *Recorder {
	cfg.defaults()

	r := &Recorder{
		queuedEventsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: promNamespace,
			Subsystem: promControllerSubsystem,
			Name:      "queued_events_total",
			Help:      "Total number of events queued.",
		}, []string{"controller", "requeue"}),

		inQueueEventDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: promNamespace,
			Subsystem: promControllerSubsystem,
			Name:      "event_in_queue_duration_total",
			Help:      "The duration of an event in the queue.",
			Buckets:   cfg.InQueueBuckets,
		}, []string{"controller"}),

		processedEventDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: promNamespace,
			Subsystem: promControllerSubsystem,
			Name:      "processed_event_duration_seconds",
			Help:      "The duration for an event to be processed.",
			Buckets:   cfg.ProcessingBuckets,
		}, []string{"controller", "success"}),
	}

	// Register metrics.
	cfg.Registerer.MustRegister(
		r.queuedEventsTotal,
		r.inQueueEventDuration,
		r.processedEventDuration)

	return r
}

// IncResourceEventQueued satisfies controller.MetricsRecorder interface.
func (r Recorder) IncResourceEventQueued(ctx context.Context, controller string, isRequeue bool) {
	r.queuedEventsTotal.WithLabelValues(controller, strconv.FormatBool(isRequeue)).Inc()
}

// ObserveResourceInQueueDuration satisfies controller.MetricsRecorder interface.
func (r Recorder) ObserveResourceInQueueDuration(ctx context.Context, controller string, queuedAt time.Time) {
	r.inQueueEventDuration.WithLabelValues(controller).
		Observe(time.Since(queuedAt).Seconds())
}

// ObserveResourceProcessingDuration satisfies controller.MetricsRecorder interface.
func (r Recorder) ObserveResourceProcessingDuration(ctx context.Context, controller string, success bool, startProcessingAt time.Time) {
	r.processedEventDuration.WithLabelValues(controller, strconv.FormatBool(success)).
		Observe(time.Since(startProcessingAt).Seconds())
}

// Check interfaces implementation.
var _ controller.MetricsRecorder = &Recorder{}
