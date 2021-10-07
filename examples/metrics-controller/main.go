package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
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
	kooperprometheus "github.com/spotahome/kooper/v2/metrics/prometheus"
)

const (
	metricsPrefix     = "metricsexample"
	metricsAddr       = ":7777"
	prometheusBackend = "prometheus"
)

var (
	metricsBackend string
)

func initFlags() error {
	fg := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fg.StringVar(&metricsBackend, "metrics-backend", "prometheus", "the metrics backend to use")
	return fg.Parse(os.Args[1:])
}

// sleep will sleep randomly from 0 to 1000 milliseconds.
func sleepRandomly() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sleepMS := r.Intn(10) * 100
	time.Sleep(time.Duration(sleepMS) * time.Millisecond)
}

// errRandomly will will err randomly.
func errRandomly() error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if r.Intn(10)%3 == 0 {
		return fmt.Errorf("random error :)")
	}
	return nil
}

// creates prometheus recorder and starts serving metrics in background.
func createPrometheusRecorder(logger log.Logger) *kooperprometheus.Recorder {
	// We could use also prometheus global registry (the default one)
	// prometheus.DefaultRegisterer instead of creating a new one
	reg := prometheus.NewRegistry()
	rec := kooperprometheus.New(kooperprometheus.Config{Registerer: reg})

	// Start serving metrics in background.
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	go func() {
		logger.Infof("serving metrics at %s", metricsAddr)
		http.ListenAndServe(metricsAddr, h)
	}()

	return rec
}

func run() error {
	// Initialize logger.
	logger := kooperlogrus.New(logrus.NewEntry(logrus.New())).
		WithKV(log.KV{"example": "metrics-controller"})

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
			return k8scli.CoreV1().Pods("").List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return k8scli.CoreV1().Pods("").Watch(context.Background(), options)
		},
	})

	// Our domain logic that will print every add/sync/update and delete event we .
	hand := controller.HandlerFunc(func(_ context.Context, obj runtime.Object) error {
		sleepRandomly()
		return errRandomly()
	})

	// Create the controller that will refresh every 30 seconds.
	cfg := &controller.Config{
		Name:                 "metricsControllerTest",
		Handler:              hand,
		Retriever:            retr,
		MetricsRecorder:      createPrometheusRecorder(logger),
		Logger:               logger,
		ProcessingJobRetries: 3,
	}
	ctrl, err := controller.New(cfg)
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}

	// Start our controller.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := ctrl.Run(ctx); err != nil {
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
