package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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
	"github.com/spotahome/kooper/controller/leaderelection"
	"github.com/spotahome/kooper/log"
	kooperlogrus "github.com/spotahome/kooper/log/logrus"
)

const (
	leaderElectionKey        = "leader-election-example-controller"
	namespaceDef             = "default"
	resyncIntervalSecondsDef = 30
)

// Flags are the flags of the program.
type Flags struct {
	ResyncIntervalSeconds int
	Namespace             string
}

// NewFlags returns the flags of the commandline.
func NewFlags() *Flags {
	flags := &Flags{}
	fl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	fl.IntVar(&flags.ResyncIntervalSeconds, "resync-interval", resyncIntervalSecondsDef, "resync seconds of the controller")
	fl.StringVar(&flags.Namespace, "namespace", namespaceDef, "kubernetes namespace where the controller is running")

	fl.Parse(os.Args[1:])

	return flags
}

func run() error {
	// Flags
	fl := NewFlags()

	// Initialize logger.
	logger := kooperlogrus.New(logrus.NewEntry(logrus.New())).
		WithKV(log.KV{"example": "leader-election-controller"})

	// Get k8s client.
	k8scfg, err := rest.InClusterConfig()
	if err != nil {
		// No in cluster? lets try locally
		kubehome := filepath.Join(homedir.HomeDir(), ".kube", "config")
		k8scfg, err = clientcmd.BuildConfigFromFlags("", kubehome)
		if err != nil {
			return fmt.Errorf("error loading kubernetes configuration: %s", err)
		}
	}
	k8scli, err := kubernetes.NewForConfig(k8scfg)
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %s", err)
	}

	// Create our retriever so the controller knows how to get/listen for pod events.
	retr := &controller.Resource{
		Object: &corev1.Pod{},
		ListerWatcher: &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return k8scli.CoreV1().Pods("").List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return k8scli.CoreV1().Pods("").Watch(options)
			},
		},
	}

	// Our domain logic that will print every add/sync/update and delete event we .
	hand := &controller.HandlerFunc{
		AddFunc: func(_ context.Context, obj runtime.Object) error {
			pod := obj.(*corev1.Pod)
			logger.Infof("Pod added: %s/%s", pod.Namespace, pod.Name)
			return nil
		},
		DeleteFunc: func(_ context.Context, s string) error {
			logger.Infof("Pod deleted: %s", s)
			return nil
		},
	}

	// Leader election service.
	lesvc, err := leaderelection.NewDefault(leaderElectionKey, fl.Namespace, k8scli, logger)
	if err != nil {
		return err
	}

	// Create the controller and run.
	cfg := &controller.Config{
		Name:          "leader-election-controller",
		Handler:       hand,
		Retriever:     retr,
		LeaderElector: lesvc,
		Logger:        logger,

		ProcessingJobRetries: 5,
		ResyncInterval:       time.Duration(fl.ResyncIntervalSeconds) * time.Second,
		ConcurrentWorkers:    1,
	}
	ctrl, err := controller.New(cfg)
	if err != nil {
		return fmt.Errorf("error creating controller: %w", err)
	}
	stopC := make(chan struct{})
	errC := make(chan error)
	go func() {
		errC <- ctrl.Run(stopC)
	}()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-errC:
		if err != nil {
			logger.Infof("controller finished with error: %s", err)
			return err
		}
		logger.Infof("controller finished successfuly")
	case s := <-sigC:
		logger.Infof("signal %s received", s)
		close(stopC)
	}

	time.Sleep(5 * time.Second)

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error executing controller: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}
