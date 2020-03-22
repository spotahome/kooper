package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/spotahome/kooper/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

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
		Handler:   newHandler(kubeCli, logger),
		Retriever: newRetriever(podTermCli),
		Logger:    logger,

		ResyncInterval: cfg.ResyncPeriod,
	})
}

func newRetriever(cli podtermk8scli.Interface) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return cli.ChaosV1alpha1().PodTerminators().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return cli.ChaosV1alpha1().PodTerminators().Watch(options)
		},
	})
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
