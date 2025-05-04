package prometheus_test

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"

	kooperprometheus "github.com/spotahome/kooper/v2/metrics/prometheus"
)

func TestPrometheusRecorder(t *testing.T) {
	tests := map[string]struct {
		cfg        kooperprometheus.Config
		addMetrics func(*kooperprometheus.Recorder)
		expMetrics []string
	}{
		"Incremeneting the total queued resource events should record the metrics.": {
			addMetrics: func(r *kooperprometheus.Recorder) {
				ctx := context.TODO()
				r.IncResourceEventQueued(ctx, "ctrl1", false)
				r.IncResourceEventQueued(ctx, "ctrl1", false)
				r.IncResourceEventQueued(ctx, "ctrl2", false)
				r.IncResourceEventQueued(ctx, "ctrl3", true)
				r.IncResourceEventQueued(ctx, "ctrl3", true)
				r.IncResourceEventQueued(ctx, "ctrl3", false)
			},
			expMetrics: []string{
				`# HELP kooper_controller_queued_events_total Total number of events queued.`,
				`# TYPE kooper_controller_queued_events_total counter`,

				`kooper_controller_queued_events_total{controller="ctrl1",requeue="false"} 2`,
				`kooper_controller_queued_events_total{controller="ctrl2",requeue="false"} 1`,
				`kooper_controller_queued_events_total{controller="ctrl3",requeue="false"} 1`,
				`kooper_controller_queued_events_total{controller="ctrl3",requeue="true"} 2`,
			},
		},

		"Observing the duration in queue of events should record the metrics.": {
			addMetrics: func(r *kooperprometheus.Recorder) {
				ctx := context.TODO()
				t0 := time.Now()
				r.ObserveResourceInQueueDuration(ctx, "ctrl1", t0.Add(-30*time.Second))
				r.ObserveResourceInQueueDuration(ctx, "ctrl1", t0.Add(-127*time.Millisecond))
				r.ObserveResourceInQueueDuration(ctx, "ctrl2", t0.Add(-70*time.Millisecond))
				r.ObserveResourceInQueueDuration(ctx, "ctrl2", t0.Add(-35*time.Millisecond))
				r.ObserveResourceInQueueDuration(ctx, "ctrl2", t0.Add(-500*time.Millisecond))
				r.ObserveResourceInQueueDuration(ctx, "ctrl2", t0.Add(-2*time.Second))
			},
			expMetrics: []string{
				`# HELP kooper_controller_event_in_queue_duration_seconds The duration of an event in the queue.`,
				`# TYPE kooper_controller_event_in_queue_duration_seconds histogram`,

				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="0.01"} 0`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="0.05"} 0`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="0.1"} 0`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="0.25"} 1`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="0.5"} 1`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="1"} 1`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="3"} 1`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="10"} 1`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="20"} 1`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="60"} 2`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="150"} 2`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="300"} 2`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="+Inf"} 2`,
				`kooper_controller_event_in_queue_duration_seconds_count{controller="ctrl1"} 2`,

				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="0.01"} 0`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="0.05"} 1`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="0.1"} 2`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="0.25"} 2`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="0.5"} 2`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="1"} 3`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="3"} 4`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="10"} 4`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="20"} 4`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="60"} 4`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="150"} 4`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="300"} 4`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl2",le="+Inf"} 4`,
				`kooper_controller_event_in_queue_duration_seconds_count{controller="ctrl2"} 4`,
			},
		},

		"Observing the duration in queue of events should record the metrics (Custom buckets).": {
			cfg: kooperprometheus.Config{
				InQueueBuckets: []float64{10, 20, 30, 50},
			},
			addMetrics: func(r *kooperprometheus.Recorder) {
				ctx := context.TODO()
				t0 := time.Now()
				r.ObserveResourceInQueueDuration(ctx, "ctrl1", t0.Add(-6*time.Second))
				r.ObserveResourceInQueueDuration(ctx, "ctrl1", t0.Add(-12*time.Second))
				r.ObserveResourceInQueueDuration(ctx, "ctrl1", t0.Add(-25*time.Second))
				r.ObserveResourceInQueueDuration(ctx, "ctrl1", t0.Add(-60*time.Second))
				r.ObserveResourceInQueueDuration(ctx, "ctrl1", t0.Add(-70*time.Second))

			},
			expMetrics: []string{
				`# HELP kooper_controller_event_in_queue_duration_seconds The duration of an event in the queue.`,
				`# TYPE kooper_controller_event_in_queue_duration_seconds histogram`,

				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="10"} 1`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="20"} 2`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="30"} 3`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="50"} 3`,
				`kooper_controller_event_in_queue_duration_seconds_bucket{controller="ctrl1",le="+Inf"} 5`,
				`kooper_controller_event_in_queue_duration_seconds_count{controller="ctrl1"} 5`,
			},
		},

		"Observing the duration of processing events should record the metrics.": {
			addMetrics: func(r *kooperprometheus.Recorder) {
				ctx := context.TODO()
				t0 := time.Now()
				r.ObserveResourceProcessingDuration(ctx, "ctrl1", true, t0.Add(-3*time.Second))
				r.ObserveResourceProcessingDuration(ctx, "ctrl1", true, t0.Add(-280*time.Millisecond))
				r.ObserveResourceProcessingDuration(ctx, "ctrl2", true, t0.Add(-7*time.Second))
				r.ObserveResourceProcessingDuration(ctx, "ctrl2", false, t0.Add(-35*time.Millisecond))
				r.ObserveResourceProcessingDuration(ctx, "ctrl2", true, t0.Add(-770*time.Millisecond))
				r.ObserveResourceProcessingDuration(ctx, "ctrl2", false, t0.Add(-17*time.Millisecond))
			},
			expMetrics: []string{
				`# HELP kooper_controller_processed_event_duration_seconds The duration for an event to be processed.`,
				`# TYPE kooper_controller_processed_event_duration_seconds histogram`,

				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="0.005"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="0.01"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="0.025"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="0.05"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="0.1"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="0.25"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="0.5"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="1"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="2.5"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="5"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="10"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="+Inf"} 2`,
				`kooper_controller_processed_event_duration_seconds_count{controller="ctrl1",success="true"} 2`,

				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="0.005"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="0.01"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="0.025"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="0.05"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="0.1"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="0.25"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="0.5"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="1"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="2.5"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="5"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="10"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="false",le="+Inf"} 2`,
				`kooper_controller_processed_event_duration_seconds_count{controller="ctrl2",success="false"} 2`,

				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="0.005"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="0.01"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="0.025"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="0.05"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="0.1"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="0.25"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="0.5"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="1"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="2.5"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="5"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="10"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl2",success="true",le="+Inf"} 2`,
				`kooper_controller_processed_event_duration_seconds_count{controller="ctrl2",success="true"} 2`,
			},
		},

		"Observing the duration of processing events should record the metrics (Custom buckets).": {
			cfg: kooperprometheus.Config{
				ProcessingBuckets: []float64{10, 20, 30, 50},
			},
			addMetrics: func(r *kooperprometheus.Recorder) {
				ctx := context.TODO()
				t0 := time.Now()
				r.ObserveResourceProcessingDuration(ctx, "ctrl1", true, t0.Add(-6*time.Second))
				r.ObserveResourceProcessingDuration(ctx, "ctrl1", true, t0.Add(-12*time.Second))
				r.ObserveResourceProcessingDuration(ctx, "ctrl1", true, t0.Add(-25*time.Second))
				r.ObserveResourceProcessingDuration(ctx, "ctrl1", true, t0.Add(-60*time.Second))
				r.ObserveResourceProcessingDuration(ctx, "ctrl1", true, t0.Add(-70*time.Second))
			},
			expMetrics: []string{
				`# HELP kooper_controller_processed_event_duration_seconds The duration for an event to be processed.`,
				`# TYPE kooper_controller_processed_event_duration_seconds histogram`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="10"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="20"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="30"} 3`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="50"} 3`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="ctrl1",success="true",le="+Inf"} 5`,
				`kooper_controller_processed_event_duration_seconds_count{controller="ctrl1",success="true"} 5`,
			},
		},

		"Registering resource queue length function should measure the size of the queue.": {
			cfg: kooperprometheus.Config{},
			addMetrics: func(r *kooperprometheus.Recorder) {
				_ = r.RegisterResourceQueueLengthFunc("ctrl1", func(_ context.Context) int { return 42 })
				_ = r.RegisterResourceQueueLengthFunc("ctrl2", func(_ context.Context) int { return 142 })
				_ = r.RegisterResourceQueueLengthFunc("ctrl3", func(_ context.Context) int { return 242 })
			},
			expMetrics: []string{
				`# HELP kooper_controller_event_queue_length Length of the controller resource queue.`,
				`# TYPE kooper_controller_event_queue_length gauge`,
				`kooper_controller_event_queue_length{controller="ctrl1"} 42`,
				`kooper_controller_event_queue_length{controller="ctrl2"} 142`,
				`kooper_controller_event_queue_length{controller="ctrl3"} 242`,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			// Create a new prometheus empty registry and a kooper prometheus recorder.
			reg := prometheus.NewRegistry()
			test.cfg.Registerer = reg
			m := kooperprometheus.New(test.cfg)

			// Add desired metrics
			test.addMetrics(m)

			// Ask prometheus for the metrics
			h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			r := httptest.NewRequest("GET", "/metrics", nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			resp := w.Result()

			// Check all metrics are present.
			body, _ := io.ReadAll(resp.Body)
			for _, expMetric := range test.expMetrics {
				assert.Contains(string(body), expMetric, "metric not present on the result of metrics service")
			}
		})
	}
}
