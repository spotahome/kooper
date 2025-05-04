package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/spotahome/kooper/v2/controller"
	"github.com/spotahome/kooper/v2/log"
	kooperlogrus "github.com/spotahome/kooper/v2/log/logrus"
)

func run() error {
	// Initialize logger.
	logger := kooperlogrus.New(logrus.NewEntry(logrus.New())).
		WithKV(log.KV{"example": "multi-resource-controller"})

	// Get k8s client.
	k8scfg, err := rest.InClusterConfig()
	if err != nil {
		// No in cluster? letr's try locally
		kubehome := filepath.Join(homedir.HomeDir(), ".kube", "config")
		k8scfg, err = clientcmd.BuildConfigFromFlags("", kubehome)
		if err != nil {
			return fmt.Errorf("error loading kubernetes configuration: %w", err)
		}
	}
	k8scli, err := kubernetes.NewForConfig(k8scfg)
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %w", err)
	}

	// Our domain logic that will print every add/sync/update and delete event we .
	hand := controller.HandlerFunc(func(_ context.Context, obj runtime.Object) error {
		dep, ok := obj.(*appsv1.Deployment)
		if ok {
			logger.Infof("Deployment added: %s/%s", dep.Namespace, dep.Name)
			return nil
		}

		st, ok := obj.(*appsv1.StatefulSet)
		if ok {
			logger.Infof("Statefulset added: %s/%s", st.Namespace, st.Name)
			return nil
		}

		return nil
	})

	const (
		retries        = 5
		resyncInterval = 45 * time.Second
		workers        = 1
	)

	// Create the controller for deployments.
	ctrlDep, err := controller.New(&controller.Config{
		Name:    "multi-resource-controller-deployments",
		Handler: hand,
		Retriever: controller.MustRetrieverFromListerWatcher(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return k8scli.AppsV1().Deployments("").List(context.Background(), options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return k8scli.AppsV1().Deployments("").Watch(context.Background(), options)
				},
			},
		),
		Logger:               logger,
		ProcessingJobRetries: retries,
		ResyncInterval:       resyncInterval,
		ConcurrentWorkers:    workers,
	})
	if err != nil {
		return fmt.Errorf("could not create deployment resource controller: %w", err)
	}

	// Create the controller for statefulsets.
	ctrlSt, err := controller.New(&controller.Config{
		Name:    "multi-resource-controller-statefulsets",
		Handler: hand,
		Retriever: controller.MustRetrieverFromListerWatcher(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return k8scli.AppsV1().StatefulSets("").List(context.Background(), options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return k8scli.AppsV1().StatefulSets("").Watch(context.Background(), options)
				},
			},
		),
		Logger:               logger,
		ProcessingJobRetries: retries,
		ResyncInterval:       resyncInterval,
		ConcurrentWorkers:    workers,
	})
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}

	// Start our controllers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errC := make(chan error)
	go func() {
		errC <- ctrlDep.Run(ctx)
	}()

	go func() {
		errC <- ctrlSt.Run(ctx)
	}()

	// Wait until one finishes.
	err = <-errC
	if err != nil {
		return fmt.Errorf("error running controllers: %w", err)
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running app: %s", err)
		os.Exit(1)
	}

	os.Exit(0)
}
