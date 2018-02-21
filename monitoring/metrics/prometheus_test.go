package metrics_test

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"

	"github.com/spotahome/kooper/monitoring/metrics"
)

func TestPrometheusMetrics(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		addMetrics func(*metrics.Prometheus)
		expMetrics []string
		expCode    int
	}{
		{
			name:      "Incrementing different kind of queued events should measure the queued events counter",
			namespace: "kooper",
			addMetrics: func(p *metrics.Prometheus) {
				p.IncResourceAddEventQueued()
				p.IncResourceDeleteEventQueued()
				p.IncResourceAddEventQueued()
				p.IncResourceAddEventQueued()
				p.IncResourceDeleteEventQueued()
				p.IncResourceDeleteEventQueued()
				p.IncResourceAddEventQueued()
			},
			expMetrics: []string{
				`kooper_controller_queued_events_total{type="add"} 4`,
				`kooper_controller_queued_events_total{type="delete"} 3`,
			},
			expCode: 200,
		},
		{
			name:      "Incrementing different kind of processed events should measure the queued events counter",
			namespace: "batman",
			addMetrics: func(p *metrics.Prometheus) {
				p.IncResourceAddEventProcessedSuccess()
				p.IncResourceAddEventProcessedError()
				p.IncResourceAddEventProcessedError()
				p.IncResourceDeleteEventProcessedSuccess()
				p.IncResourceDeleteEventProcessedSuccess()
				p.IncResourceDeleteEventProcessedSuccess()
				p.IncResourceDeleteEventProcessedError()
				p.IncResourceDeleteEventProcessedError()
				p.IncResourceDeleteEventProcessedError()
				p.IncResourceDeleteEventProcessedError()

			},
			expMetrics: []string{
				`batman_controller_processed_events_total{type="add"} 1`,
				`batman_controller_processed_events_errors_total{type="add"} 2`,
				`batman_controller_processed_events_total{type="delete"} 3`,
				`batman_controller_processed_events_errors_total{type="delete"} 4`,
			},
			expCode: 200,
		},
		{
			name:      "Measuring the duration of processed Add events return the correct buckets.",
			namespace: "spiderman",
			addMetrics: func(p *metrics.Prometheus) {
				now := time.Now()
				p.ObserveDurationResourceAddEventProcessedError(now.Add(-2 * time.Millisecond))
				p.ObserveDurationResourceAddEventProcessedError(now.Add(-3 * time.Millisecond))
				p.ObserveDurationResourceAddEventProcessedError(now.Add(-11 * time.Millisecond))
				p.ObserveDurationResourceAddEventProcessedError(now.Add(-280 * time.Millisecond))
				p.ObserveDurationResourceAddEventProcessedError(now.Add(-1 * time.Second))
				p.ObserveDurationResourceAddEventProcessedError(now.Add(-5 * time.Second))
				p.ObserveDurationResourceAddEventProcessedSuccess(now.Add(-110 * time.Millisecond))
				p.ObserveDurationResourceAddEventProcessedSuccess(now.Add(-560 * time.Millisecond))
				p.ObserveDurationResourceAddEventProcessedSuccess(now.Add(-4 * time.Second))
				p.ObserveDurationResourceAddEventProcessedSuccess(now.Add(-7 * time.Second))
				p.ObserveDurationResourceAddEventProcessedSuccess(now.Add(-12 * time.Second))
				p.ObserveDurationResourceAddEventProcessedSuccess(now.Add(-30 * time.Second))
			},
			expMetrics: []string{
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="0.005"} 2`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="0.01"} 2`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="0.025"} 3`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="0.05"} 3`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="0.1"} 3`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="0.25"} 3`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="0.5"} 4`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="1"} 4`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="2.5"} 5`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="5"} 5`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="10"} 6`,
				`spiderman_controller_processed_events_error_duration_seconds_bucket{type="add",le="+Inf"} 6`,
				`spiderman_controller_processed_events_error_duration_seconds_count{type="add"} 6`,

				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="0.005"} 0`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="0.01"} 0`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="0.025"} 0`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="0.05"} 0`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="0.1"} 0`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="0.25"} 1`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="0.5"} 1`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="1"} 2`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="2.5"} 2`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="5"} 3`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="10"} 4`,
				`spiderman_controller_processed_events_duration_seconds_bucket{type="add",le="+Inf"} 6`,
				`spiderman_controller_processed_events_duration_seconds_count{type="add"} 6`,
			},
			expCode: 200,
		},
		{
			name:      "Measuring the duration of processed Delete events return the correct buckets.",
			namespace: "deadpool",
			addMetrics: func(p *metrics.Prometheus) {
				now := time.Now()
				p.ObserveDurationResourceDeleteEventProcessedError(now.Add(-2 * time.Millisecond))
				p.ObserveDurationResourceDeleteEventProcessedError(now.Add(-3 * time.Millisecond))
				p.ObserveDurationResourceDeleteEventProcessedError(now.Add(-11 * time.Millisecond))
				p.ObserveDurationResourceDeleteEventProcessedError(now.Add(-280 * time.Millisecond))
				p.ObserveDurationResourceDeleteEventProcessedError(now.Add(-1 * time.Second))
				p.ObserveDurationResourceDeleteEventProcessedError(now.Add(-5 * time.Second))
				p.ObserveDurationResourceDeleteEventProcessedSuccess(now.Add(-110 * time.Millisecond))
				p.ObserveDurationResourceDeleteEventProcessedSuccess(now.Add(-560 * time.Millisecond))
				p.ObserveDurationResourceDeleteEventProcessedSuccess(now.Add(-4 * time.Second))
				p.ObserveDurationResourceDeleteEventProcessedSuccess(now.Add(-7 * time.Second))
				p.ObserveDurationResourceDeleteEventProcessedSuccess(now.Add(-12 * time.Second))
				p.ObserveDurationResourceDeleteEventProcessedSuccess(now.Add(-30 * time.Second))
			},
			expMetrics: []string{
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="0.005"} 2`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="0.01"} 2`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="0.025"} 3`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="0.05"} 3`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="0.1"} 3`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="0.25"} 3`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="0.5"} 4`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="1"} 4`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="2.5"} 5`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="5"} 5`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="10"} 6`,
				`deadpool_controller_processed_events_error_duration_seconds_bucket{type="delete",le="+Inf"} 6`,
				`deadpool_controller_processed_events_error_duration_seconds_count{type="delete"} 6`,

				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="0.005"} 0`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="0.01"} 0`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="0.025"} 0`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="0.05"} 0`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="0.1"} 0`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="0.25"} 1`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="0.5"} 1`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="1"} 2`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="2.5"} 2`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="5"} 3`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="10"} 4`,
				`deadpool_controller_processed_events_duration_seconds_bucket{type="delete",le="+Inf"} 6`,
				`deadpool_controller_processed_events_duration_seconds_count{type="delete"} 6`,
			},
			expCode: 200,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Create a new prometheus empty registry and a kooper prometheus recorder.
			reg := prometheus.NewRegistry()
			m := metrics.NewPrometheus(test.namespace, reg)

			// Add desired metrics
			test.addMetrics(m)

			// Ask prometheus for the metrics
			h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			r := httptest.NewRequest("GET", "/metrics", nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			resp := w.Result()

			// Check all metrics are present.
			if assert.Equal(test.expCode, resp.StatusCode) {
				body, _ := ioutil.ReadAll(resp.Body)
				for _, expMetric := range test.expMetrics {
					assert.Contains(string(body), expMetric, "metric not present on the result of metrics service")
				}
			}
		})
	}
}
