package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
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

var (
	concurrentWorkers int
	sleepMS           int
	intervalS         int
	retries           int
)

func initFlags() error {
	fg := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fg.IntVar(&concurrentWorkers, "concurrency", 3, "The number of concurrent event handling")
	fg.IntVar(&sleepMS, "sleep-ms", 25, "The number of milliseconds to sleep on each event handling")
	fg.IntVar(&intervalS, "interval-s", 45, "The number of seconds to for reconciliation loop intervals")
	fg.IntVar(&retries, "retries", 3, "The number of retries in case of error")

	err := fg.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	if concurrentWorkers < 1 {
		concurrentWorkers = 1
	}

	if sleepMS < 1 {
		sleepMS = 25
	}

	if intervalS < 1 {
		intervalS = 300
	}
	if retries < 0 {
		retries = 0
	}

	return nil
}

func sleep() {
	time.Sleep(time.Duration(sleepMS) * time.Millisecond)
}

func run() error {
	// Initialize logger.
	logger := kooperlogrus.New(logrus.NewEntry(logrus.New())).
		WithKV(log.KV{"example": "controller-concurrency-handling"})

	// Init flags.
	if err := initFlags(); err != nil {
		return fmt.Errorf("error parsing arguments: %w", err)
	}

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

	// Create our retriever so the controller knows how to get/listen for pod events.
	retr := controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return k8scli.CoreV1().Pods("").List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return k8scli.CoreV1().Pods("").Watch(options)
		},
	})

	// Our domain logic that will print every add/sync/update and delete event we.
	hand := controller.HandlerFunc(func(_ context.Context, obj runtime.Object) error {
		pod := obj.(*corev1.Pod)
		sleep()
		logger.Infof("Pod added: %s/%s", pod.Namespace, pod.Name)
		return nil
	})

	// Create the controller.
	cfg := &controller.Config{
		Name:      "controller-concurrency-handling",
		Handler:   hand,
		Retriever: retr,
		Logger:    logger,

		ProcessingJobRetries: retries,
		ResyncInterval:       time.Duration(intervalS) * time.Second,
		ConcurrentWorkers:    concurrentWorkers,
	}
	ctrl, err := controller.New(cfg)
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}

	// Start our controller.
	stopC := make(chan struct{})
	if err := ctrl.Run(stopC); err != nil {
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
