package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/spotahome/kooper/controller/leaderelection"
	"github.com/spotahome/kooper/log"
	"github.com/spotahome/kooper/monitoring/metrics"
)

var (
	// ErrControllerNotValid will be used when the controller has invalid configuration.
	ErrControllerNotValid = errors.New("controller not valid")
)

// Defaults.
const (
	defResyncInterval       = 3 * time.Minute
	defConcurrentWorkers    = 3
	defProcessingJobRetries = 3
)

// Config is the controller configuration.
type Config struct {
	// Handler is the controller handler.
	Handler Handler
	// Retriever is the controller retriever.
	Retriever Retriever
	// Leader elector will be used to use only one instance, if no set it will be
	// leader election will be ignored
	LeaderElector leaderelection.Runner
	// MetricsRecorder will record the controller metrics.
	MetricRecorder metrics.Recorder
	// Logger will log messages of the controller.
	Logger log.Logger

	// name of the controller.
	Name string
	// ConcurrentWorkers is the number of concurrent workers the controller will have running processing events.
	ConcurrentWorkers int
	// ResyncInterval is the interval the controller will process all the selected resources.
	ResyncInterval time.Duration
	// ProcessingJobRetries is the number of times the job will try to reprocess the event before returning a real error.
	ProcessingJobRetries int
}

func (c *Config) setDefaults() error {
	if c.Name == "" {
		return fmt.Errorf("a controller name is required")
	}

	if c.Handler == nil {
		return fmt.Errorf("a handler is required")
	}

	if c.Retriever == nil {
		return fmt.Errorf("a retriever is required")
	}

	if c.Logger == nil {
		c.Logger = &log.Std{}
		c.Logger.Warningf("no logger specified, fallback to default logger, to disable logging use a explicit Noop logger")
	}

	if c.MetricRecorder == nil {
		c.MetricRecorder = metrics.Dummy
		c.Logger.Warningf("no metrics recorder specified, disabling metrics")
	}

	if c.ConcurrentWorkers <= 0 {
		c.ConcurrentWorkers = defConcurrentWorkers
	}

	if c.ResyncInterval <= 0 {
		c.ResyncInterval = defResyncInterval
	}

	if c.ProcessingJobRetries <= 0 {
		c.ProcessingJobRetries = defProcessingJobRetries
	}

	return nil
}

// generic controller is a controller that can be used to create different kind of controllers.
type generic struct {
	queue     workqueue.RateLimitingInterface // queue will have the jobs that the controller will get and send to handlers.
	informer  cache.SharedIndexInformer       // informer will notify be inform us about resource changes.
	handler   Handler                         // handler is where the logic of resource processing.
	running   bool
	runningMu sync.Mutex
	cfg       Config
	metrics   metrics.Recorder
	leRunner  leaderelection.Runner
	logger    log.Logger
}

// New creates a new controller that can be configured using the cfg parameter.
func New(cfg *Config) (Controller, error) {
	// Sets the required default configuration.
	err := cfg.setDefaults()
	if err != nil {
		return nil, fmt.Errorf("could no create controller: %w: %v", ErrControllerNotValid, err)
	}

	// Create the queue that will have our received job changes. It's rate limited so we don't have problems when
	// a job processing errors every time is processed in a loop.
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// store is the internal cache where objects will be store.
	store := cache.Indexers{}
	informer := cache.NewSharedIndexInformer(cfg.Retriever.GetListerWatcher(), cfg.Retriever.GetObject(), cfg.ResyncInterval, store)

	// Set up our informer event handler.
	// Objects are already in our local store. Add only keys/jobs on the queue so they can bre processed
	// afterwards.
	informer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
				cfg.MetricRecorder.IncResourceEventQueued(cfg.Name, metrics.AddEvent)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
				cfg.MetricRecorder.IncResourceEventQueued(cfg.Name, metrics.AddEvent)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
				cfg.MetricRecorder.IncResourceEventQueued(cfg.Name, metrics.DeleteEvent)
			}
		},
	}, cfg.ResyncInterval)

	// Create our generic controller object.
	return &generic{
		queue:    queue,
		informer: informer,
		logger:   cfg.Logger,
		metrics:  cfg.MetricRecorder,
		handler:  cfg.Handler,
		leRunner: cfg.LeaderElector,
		cfg:      *cfg,
	}, nil
}

func (g *generic) isRunning() bool {
	g.runningMu.Lock()
	defer g.runningMu.Unlock()
	return g.running
}

func (g *generic) setRunning(running bool) {
	g.runningMu.Lock()
	defer g.runningMu.Unlock()
	g.running = running
}

// Run will run the controller.
func (g *generic) Run(stopC <-chan struct{}) error {
	// Check if leader election is required.
	if g.leRunner != nil {
		return g.leRunner.Run(func() error {
			return g.run(stopC)
		})
	}

	return g.run(stopC)
}

// run is the real run of the controller.
func (g *generic) run(stopC <-chan struct{}) error {
	if g.isRunning() {
		return fmt.Errorf("controller already running")
	}

	g.logger.Infof("starting controller")
	// Set state of controller.
	g.setRunning(true)
	defer g.setRunning(false)

	// Shutdown when Run is stopped so we can process the last items and the queue doesn't
	// accept more jobs.
	defer g.queue.ShutDown()

	// Run the informer so it starts listening to resource events.
	go g.informer.Run(stopC)

	// Wait until our store, jobs... stuff is synced (first list on resource, resources on store and jobs on queue).
	if !cache.WaitForCacheSync(stopC, g.informer.HasSynced) {
		return fmt.Errorf("timed out waiting for caches to sync")
	}

	// Start our resource processing worker, if finishes then restart the worker. The workers should
	// not end.
	for i := 0; i < g.cfg.ConcurrentWorkers; i++ {
		go func() {
			wait.Until(g.runWorker, time.Second, stopC)
		}()
	}

	// Until will be running our workers in a continuous way (and re run if they fail). But
	// when stop signal is received we must stop.
	<-stopC
	g.logger.Infof("stopping controller")

	return nil
}

// runWorker will start a processing loop on event queue.
func (g *generic) runWorker() {
	for {
		// Process newxt queue job, if needs to stop processing it will return true.
		if g.getAndProcessNextJob() {
			break
		}
	}
}

// getAndProcessNextJob job will process the next job of the queue job and returns if
// it needs to stop processing.
func (g *generic) getAndProcessNextJob() bool {
	// Get next job.
	nextJob, exit := g.queue.Get()
	if exit {
		return true
	}
	defer g.queue.Done(nextJob)
	key := nextJob.(string)

	// Process the job. If errors then enqueue again.
	ctx := context.Background()
	if err := g.processJob(ctx, key); err == nil {
		g.queue.Forget(key)
	} else if g.queue.NumRequeues(key) < g.cfg.ProcessingJobRetries {
		// Job processing failed, requeue.
		g.logger.Warningf("error processing %s job (requeued): %v", key, err)
		g.queue.AddRateLimited(key)
		g.metrics.IncResourceEventQueued(g.cfg.Name, metrics.RequeueEvent)
	} else {
		g.logger.Errorf("Error processing %s: %v", key, err)
		g.queue.Forget(key)
	}

	return false
}

// processJob is where the real processing logic of the item is.
func (g *generic) processJob(ctx context.Context, key string) error {
	// Get the object
	obj, exists, err := g.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	// handle the object.
	if !exists { // Deleted resource from the cache.
		return g.handleDelete(ctx, key)
	}

	return g.handleAdd(ctx, key, obj.(runtime.Object))
}

func (g *generic) handleAdd(ctx context.Context, objKey string, obj runtime.Object) error {
	start := time.Now()
	g.metrics.IncResourceEventProcessed(g.cfg.Name, metrics.AddEvent)
	defer func() {
		g.metrics.ObserveDurationResourceEventProcessed(g.cfg.Name, metrics.AddEvent, start)
	}()

	// Handle the job.
	if err := g.handler.Add(ctx, obj); err != nil {
		g.metrics.IncResourceEventProcessedError(g.cfg.Name, metrics.AddEvent)
		return err
	}
	return nil
}

func (g *generic) handleDelete(ctx context.Context, objKey string) error {
	start := time.Now()
	g.metrics.IncResourceEventProcessed(g.cfg.Name, metrics.DeleteEvent)
	defer func() {
		g.metrics.ObserveDurationResourceEventProcessed(g.cfg.Name, metrics.DeleteEvent, start)
	}()

	// Handle the job.
	if err := g.handler.Delete(ctx, objKey); err != nil {
		g.metrics.IncResourceEventProcessedError(g.cfg.Name, metrics.DeleteEvent)
		return err
	}
	return nil
}
