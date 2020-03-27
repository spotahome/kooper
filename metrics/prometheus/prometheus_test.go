package prometheus_test

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"

	"github.com/spotahome/kooper/controller"
	kooperprometheus "github.com/spotahome/kooper/metrics/prometheus"
)

func TestPrometheusRecorder(t *testing.T) {
	controllerID := "test"

	tests := []struct {
		name       string
		addMetrics func(*kooperprometheus.Recorder)
		expMetrics []string
		expCode    int
	}{
		{
			name: "Incrementing different kind of queued events should measure the queued events counter",
			addMetrics: func(r *kooperprometheus.Recorder) {
				r.IncResourceEventQueued(context.TODO(), controllerID, controller.AddEvent)
				r.IncResourceEventQueued(context.TODO(), controllerID, controller.AddEvent)
				r.IncResourceEventQueued(context.TODO(), controllerID, controller.AddEvent)
				r.IncResourceEventQueued(context.TODO(), controllerID, controller.AddEvent)
				r.IncResourceEventQueued(context.TODO(), controllerID, controller.DeleteEvent)
				r.IncResourceEventQueued(context.TODO(), controllerID, controller.DeleteEvent)
				r.IncResourceEventQueued(context.TODO(), controllerID, controller.DeleteEvent)
			},
			expMetrics: []string{
				`kooper_controller_queued_events_total{controller="test",type="add"} 4`,
				`kooper_controller_queued_events_total{controller="test",type="delete"} 3`,
			},
			expCode: 200,
		},
		{
			name: "Incrementing different kind of processed events should measure the processed events counter",
			addMetrics: func(r *kooperprometheus.Recorder) {
				r.IncResourceEventProcessed(context.TODO(), controllerID, controller.AddEvent)
				r.IncResourceEventProcessedError(context.TODO(), controllerID, controller.AddEvent)
				r.IncResourceEventProcessedError(context.TODO(), controllerID, controller.AddEvent)
				r.IncResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent)
				r.IncResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent)
				r.IncResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent)
				r.IncResourceEventProcessedError(context.TODO(), controllerID, controller.DeleteEvent)
				r.IncResourceEventProcessedError(context.TODO(), controllerID, controller.DeleteEvent)
				r.IncResourceEventProcessedError(context.TODO(), controllerID, controller.DeleteEvent)
				r.IncResourceEventProcessedError(context.TODO(), controllerID, controller.DeleteEvent)

			},
			expMetrics: []string{
				`kooper_controller_processed_events_total{controller="test",type="add"} 1`,
				`kooper_controller_processed_event_errors_total{controller="test",type="add"} 2`,
				`kooper_controller_processed_events_total{controller="test",type="delete"} 3`,
				`kooper_controller_processed_event_errors_total{controller="test",type="delete"} 4`,
			},
			expCode: 200,
		},
		{
			name: "Measuring the duration of processed events return the correct buckets.",
			addMetrics: func(r *kooperprometheus.Recorder) {
				now := time.Now()
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.AddEvent, now.Add(-2*time.Millisecond))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.AddEvent, now.Add(-3*time.Millisecond))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.AddEvent, now.Add(-11*time.Millisecond))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.AddEvent, now.Add(-280*time.Millisecond))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.AddEvent, now.Add(-1*time.Second))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.AddEvent, now.Add(-5*time.Second))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent, now.Add(-110*time.Millisecond))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent, now.Add(-560*time.Millisecond))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent, now.Add(-4*time.Second))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent, now.Add(-7*time.Second))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent, now.Add(-12*time.Second))
				r.ObserveDurationResourceEventProcessed(context.TODO(), controllerID, controller.DeleteEvent, now.Add(-30*time.Second))
			},
			expMetrics: []string{
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="0.005"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="0.01"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="0.025"} 3`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="0.05"} 3`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="0.1"} 3`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="0.25"} 3`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="0.5"} 4`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="1"} 4`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="2.5"} 5`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="5"} 5`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="10"} 6`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="add",le="+Inf"} 6`,
				`kooper_controller_processed_event_duration_seconds_count{controller="test",type="add"} 6`,

				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="0.005"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="0.01"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="0.025"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="0.05"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="0.1"} 0`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="0.25"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="0.5"} 1`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="1"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="2.5"} 2`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="5"} 3`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="10"} 4`,
				`kooper_controller_processed_event_duration_seconds_bucket{controller="test",type="delete",le="+Inf"} 6`,
				`kooper_controller_processed_event_duration_seconds_count{controller="test",type="delete"} 6`,
			},
			expCode: 200,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Create a new prometheus empty registry and a kooper prometheus recorder.
			reg := prometheus.NewRegistry()
			m := kooperprometheus.New(kooperprometheus.Config{
				Registerer: reg,
			})

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
