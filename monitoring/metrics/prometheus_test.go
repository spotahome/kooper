package metrics_test

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

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
