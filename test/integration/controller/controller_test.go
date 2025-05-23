//go:build integration
// +build integration

package controller_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/spotahome/kooper/v2/controller"
	"github.com/spotahome/kooper/v2/log"
	"github.com/spotahome/kooper/v2/test/integration/helper/cli"
	"github.com/spotahome/kooper/v2/test/integration/helper/prepare"
)

// TestControllerHandleEvents will test the controller receives the resources list and watch
// events are received and handled correctly.
func TestControllerHandleEvents(t *testing.T) {
	tests := []struct {
		name             string
		addServices      []*corev1.Service
		updateServices   []string
		expAddedServices []string
	}{
		{
			name: "If a controller is watching services it should react to the service change events.",
			addServices: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "svc1"},
					Spec: corev1.ServiceSpec{
						Type: "ClusterIP",
						Ports: []corev1.ServicePort{
							{Name: "port1", Port: 8080},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "svc2"},
					Spec: corev1.ServiceSpec{
						Type: "ClusterIP",
						Ports: []corev1.ServicePort{
							{Name: "port1", Port: 8080},
						},
					},
				},
			},
			updateServices:   []string{"svc1"},
			expAddedServices: []string{"svc1", "svc2", "svc1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			resync := 30 * time.Second
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var gotAddedServices []string

			// Create the kubernetes client.
			k8scli, err := cli.GetK8sClient("")

			require.NoError(err, "kubernetes client is required")

			// Prepare the environment on the cluster.
			prep := prepare.New(k8scli, t)
			prep.SetUp()
			defer prep.TearDown()

			// Create the retriever.
			rt := controller.MustRetrieverFromListerWatcher(cache.NewListWatchFromClient(k8scli.CoreV1().RESTClient(), "services", prep.Namespace().Name, fields.Everything()))

			// Call times are the number of times the handler should be called before sending the termination signal.
			stopCallTimes := len(test.addServices) + len(test.updateServices)
			calledTimes := 0
			var mx sync.Mutex

			// Create the handler.
			hl := controller.HandlerFunc(func(_ context.Context, obj runtime.Object) error {
				mx.Lock()
				calledTimes++
				mx.Unlock()

				svc := obj.(*corev1.Service)
				gotAddedServices = append(gotAddedServices, svc.Name)
				if calledTimes >= stopCallTimes {
					cancel()
				}
				return nil
			})

			// Create a Pod controller.
			cfg := &controller.Config{
				Name:           "test-controller",
				Handler:        hl,
				Retriever:      rt,
				Logger:         log.Dummy,
				ResyncInterval: resync,
			}
			ctrl, err := controller.New(cfg)
			require.NoError(err, "controller is required, can't have error on creation")
			go func() {
				err := ctrl.Run(ctx)
				require.NoError(err)
			}()

			// Create the required services.
			for _, svc := range test.addServices {
				_, err := k8scli.CoreV1().Services(prep.Namespace().Name).Create(context.Background(), svc, metav1.CreateOptions{})
				assert.NoError(err)
				time.Sleep(1 * time.Second)
			}

			for _, svc := range test.updateServices {
				origSvc, err := k8scli.CoreV1().Services(prep.Namespace().Name).Get(context.Background(), svc, metav1.GetOptions{})
				if assert.NoError(err) {
					// Change something
					origSvc.Spec.Ports = append(origSvc.Spec.Ports, corev1.ServicePort{Name: "updateport", Port: 9876})
					_, err := k8scli.CoreV1().Services(prep.Namespace().Name).Update(context.Background(), origSvc, metav1.UpdateOptions{})
					assert.NoError(err)
					time.Sleep(1 * time.Second)
				}
			}

			// Wait until we have finished.
			select {
			// Timeout.
			case <-time.After(20 * time.Second):
			// Finished.
			case <-ctx.Done():
			}

			// Check.
			assert.Equal(test.expAddedServices, gotAddedServices)
		})
	}
}
