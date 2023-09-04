package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/spotahome/kooper/v2/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	chaosv1alpha1 "github.com/spotahome/kooper/examples/pod-terminator-operator/v2/apis/chaos/v1alpha1"
	podtermk8scli "github.com/spotahome/kooper/examples/pod-terminator-operator/v2/client/k8s/clientset/versioned"
	"github.com/spotahome/kooper/examples/pod-terminator-operator/v2/log"
	"github.com/spotahome/kooper/examples/pod-terminator-operator/v2/service/chaos"
)

// Config is the controller configuration.
type Config struct {
	// ResyncPeriod is the resync period of the operator.
	ResyncPeriod time.Duration
}

// New returns pod terminator operator.
func New(cfg Config, podTermCli podtermk8scli.Interface, kubeCli kubernetes.Interface, logger log.Logger) (controller.Controller, error) {
	return controller.New(&controller.Config{
		Name:      "pod-terminator",
		Handler:   newHandler(kubeCli, podTermCli, logger),
		Retriever: newRetriever(podTermCli),
		Logger:    logger,

		ResyncInterval: cfg.ResyncPeriod,
	})
}

func newRetriever(cli podtermk8scli.Interface) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return cli.ChaosV1alpha1().PodTerminators().List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return cli.ChaosV1alpha1().PodTerminators().Watch(context.Background(), options)
		},
	})
}

func newHandler(k8sCli kubernetes.Interface, ptCli podtermk8scli.Interface, logger log.Logger) controller.Handler {
	const finalizer = "finalizer.chaos.spotahome.com/podKiller"
	chaossvc := chaos.NewChaos(k8sCli, logger)

	return controller.HandlerFunc(func(ctx context.Context, obj runtime.Object) error {
		pt, ok := obj.(*chaosv1alpha1.PodTerminator)
		if !ok {
			return fmt.Errorf("%v is not a pod terminator object", obj.GetObjectKind())
		}

		switch {
		// Handle deletion and remove finalizer.
		case !pt.DeletionTimestamp.IsZero() && stringPresentInSlice(pt.Finalizers, finalizer):
			logger.Infof("handling pod termination deletion...")
			err := chaossvc.DeletePodTerminator(pt.ObjectMeta.Name)
			if err != nil {
				return fmt.Errorf("could not handle PodTerminator deletion: %w", err)
			}

			pt.Finalizers = removeStringFromSlice(pt.Finalizers, finalizer)
			_, err = ptCli.ChaosV1alpha1().PodTerminators().Update(ctx, pt, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("could not update pod terminator: %w", err)
			}

			return nil

		// Deletion already handled, don't do anything.
		case !pt.DeletionTimestamp.IsZero() && !stringPresentInSlice(pt.Finalizers, finalizer):
			logger.Infof("handling pod termination deletion already handled, skipping...")
			return nil

		// Add finalizer to the object.
		case pt.DeletionTimestamp.IsZero() && !stringPresentInSlice(pt.Finalizers, finalizer):
			pt.Finalizers = append(pt.Finalizers, finalizer)
			_, err := ptCli.ChaosV1alpha1().PodTerminators().Update(ctx, pt, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("could not update pod termiantor: %w", err)
			}
		}

		// Handle.
		return chaossvc.EnsurePodTerminator(pt)
	})
}

func stringPresentInSlice(ss []string, s string) bool {
	for _, f := range ss {
		if f == s {
			return true
		}
	}
	return false
}

func removeStringFromSlice(ss []string, s string) []string {
	for i, f := range ss {
		if f == s {
			return append(ss[:i], ss[i+1:]...)
		}
	}
	return ss
}
