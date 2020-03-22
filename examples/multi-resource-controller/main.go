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

	"github.com/spotahome/kooper/controller"
	"github.com/spotahome/kooper/log"
	kooperlogrus "github.com/spotahome/kooper/log/logrus"
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

	// Create our retriever so the controller knows how to get/listen for deployments and statefulsets.
	retr, err := controller.NewMultiRetriever(
		controller.MustRetrieverFromListerWatcher(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return k8scli.AppsV1().Deployments("").List(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return k8scli.AppsV1().Deployments("").Watch(options)
				},
			},
		),
		controller.MustRetrieverFromListerWatcher(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return k8scli.AppsV1().StatefulSets("").List(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return k8scli.AppsV1().StatefulSets("").Watch(options)
				},
			},
		),
	)

	if err != nil {
		return fmt.Errorf("could not create a multi retriever: %w", err)
	}

	// Our domain logic that will print every add/sync/update and delete event we .
	hand := &controller.HandlerFunc{
		AddFunc: func(_ context.Context, obj runtime.Object) error {
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
		},
		DeleteFunc: func(_ context.Context, s string) error {
			return nil
		},
	}

	// Create the controller with custom configuration.
	cfg := &controller.Config{
		Name:      "multi-resource-controller",
		Handler:   hand,
		Retriever: retr,
		Logger:    logger,

		ProcessingJobRetries: 5,
		ResyncInterval:       45 * time.Second,
		ConcurrentWorkers:    1,
	}
	ctrl, err := controller.New(cfg)
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}

	// Start our controller.
	stopC := make(chan struct{})
	err = ctrl.Run(stopC)
	if err != nil {
		return fmt.Errorf("error running controller: %w", err)
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
