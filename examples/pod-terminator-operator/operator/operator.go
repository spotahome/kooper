package operator

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/runtime"
	"github.com/spotahome/kooper/controller"

	chaosv1alpha1 "github.com/spotahome/kooper/examples/pod-terminator-operator/apis/chaos/v1alpha1"
	podtermk8scli "github.com/spotahome/kooper/examples/pod-terminator-operator/client/k8s/clientset/versioned"
	"github.com/spotahome/kooper/examples/pod-terminator-operator/log"
	"github.com/spotahome/kooper/examples/pod-terminator-operator/service/chaos"
)

// Config is the controller configuration.
type Config struct {
	// ResyncPeriod is the resync period of the operator.
	ResyncPeriod time.Duration
}


// New returns pod terminator operator.
func New(cfg Config, podTermCli podtermk8scli.Interface, kubeCli kubernetes.Interface, logger log.Logger) (controller.Controller, error) {
	return controller.New(&controller.Config{
		Handler: newHandler(kubeCli, logger),
		Retriever: newRetriever(podTermCli),
		Logger: logger,

		ResyncInterval: cfg.ResyncPeriod,
	})
}

type retriever struct {
	podTermCli podtermk8scli.Interface
}

func newRetriever(podTermCli podtermk8scli.Interface) *retriever {
	return &retriever{
		podTermCli: podTermCli,
	}
}

// GetListerWatcher satisfies resource.crd interface (and retrieve.Retriever).
func (r retriever) GetListerWatcher() cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return r.podTermCli.ChaosV1alpha1().PodTerminators().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return r.podTermCli.ChaosV1alpha1().PodTerminators().Watch(options)
		},
	}
}

// GetObject satisfies resource.crd interface (and retrieve.Retriever).
func (r retriever) GetObject() runtime.Object {
	return &chaosv1alpha1.PodTerminator{}
}

type handler struct {
	chaosService chaos.Syncer
	logger       log.Logger
}

func newHandler(k8sCli kubernetes.Interface, logger log.Logger) *handler {
	return &handler{
		chaosService: chaos.NewChaos(k8sCli, logger),
		logger:       logger,
	}
}

func (h handler) Add(_ context.Context, obj runtime.Object) error {
	pt, ok := obj.(*chaosv1alpha1.PodTerminator)
	if !ok {
		return fmt.Errorf("%v is not a pod terminator object", obj.GetObjectKind())
	}

	return h.chaosService.EnsurePodTerminator(pt)
}


func (h handler) Delete(_ context.Context, name string) error {
	return h.chaosService.DeletePodTerminator(name)
}
