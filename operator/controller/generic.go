package controller

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/spotahome/kooper/log"
	"github.com/spotahome/kooper/monitoring/metrics"
	"github.com/spotahome/kooper/operator/controller/leaderelection"
	"github.com/spotahome/kooper/operator/handler"
	"github.com/spotahome/kooper/operator/retrieve"
)

// generic controller is a controller that can be used to create different kind of controllers.
type generic struct {
	queue       workqueue.RateLimitingInterface // queue will have the jobs that the controller will get and send to handlers.
	informer    cache.SharedIndexInformer       // informer will notify be inform us about resource changes.
	handler     handler.Handler                 // handler is where the logic of resource processing.
	handlerName string                          // handlerName will be used to identify and give more insight about metrics.
	running     bool
	runningMu   sync.Mutex
	cfg         Config
	metrics     metrics.Recorder
	leRunner    leaderelection.Runner
	logger      log.Logger
}

// NewSequential creates a new controller that will process the received events sequentially.
// This constructor is just a wrapper to help bootstrapping default sequential controller.
func NewSequential(resync time.Duration, handler handler.Handler, retriever retrieve.Retriever, metricRecorder metrics.Recorder, logger log.Logger) Controller {
	cfg := &Config{
		ConcurrentWorkers: 1,
		ResyncInterval:    resync,
	}
	return New(cfg, handler, retriever, nil, metricRecorder, logger)
}

// NewConcurrent creates a new controller that will process the received events concurrently.
// This constructor is just a wrapper to help bootstrapping default concurrent controller.
func NewConcurrent(concurrentWorkers int, resync time.Duration, handler handler.Handler, retriever retrieve.Retriever, metricRecorder metrics.Recorder, logger log.Logger) (Controller, error) {
	if concurrentWorkers < 2 {
		return nil, fmt.Errorf("%d is not a valid concurrency workers ammount for a concurrent controller", concurrentWorkers)
	}

	cfg := &Config{
		ConcurrentWorkers: concurrentWorkers,
		ResyncInterval:    resync,
	}
	return New(cfg, handler, retriever, nil, metricRecorder, logger), nil
}

// New creates a new controller that can be configured using the cfg parameter.
func New(cfg *Config, handler handler.Handler, retriever retrieve.Retriever, leaderElector leaderelection.Runner, metricRecorder metrics.Recorder, logger log.Logger) Controller {
	// Sets the required default configuration.
	cfg.setDefaults()

	// Default logger.
	if logger == nil {
		logger = &log.Std{}
		logger.Warningf("no logger specified, fallback to default logger, to disable logging use dummy logger")
	}

	// Default metrics recorder.
	if metricRecorder == nil {
		metricRecorder = metrics.Dummy
		logger.Warningf("no metrics recorder specified, disabling metrics")
	}

	// Get a handler name for the metrics based on the type of the handler.
	handlerName := reflect.TypeOf(handler).String()

	// Create the queue that will have our received job changes. It's rate limited so we don't have problems when
	// a job processing errors every time is processed in a loop.
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// store is the internal cache where objects will be store.
	store := cache.Indexers{}
	informer := cache.NewSharedIndexInformer(retriever.GetListerWatcher(), retriever.GetObject(), cfg.ResyncInterval, store)

	// Objects are already in our local store. Add only keys/jobs on the queue so they can bre processed
	// afterwards.
	informer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
				metricRecorder.IncResourceAddEventQueued(handlerName)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
				metricRecorder.IncResourceAddEventQueued(handlerName)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
				metricRecorder.IncResourceDeleteEventQueued(handlerName)
			}
		},
	}, cfg.ResyncInterval)

	return &generic{
		queue:       queue,
		informer:    informer,
		logger:      logger,
		metrics:     metricRecorder,
		handler:     handler,
		handlerName: handlerName,
		leRunner:    leaderElector,
		cfg:         *cfg,
	}
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
	key, exit := g.queue.Get()
	if exit {
		return true
	}
	objKey := key.(string)
	defer g.queue.Done(objKey)

	// Process the job. If errors then enqueue again.
	if err := g.processJob(objKey); err == nil {
		g.queue.Forget(objKey)
	} else if g.queue.NumRequeues(objKey) < g.cfg.ProcessingJobRetries {
		// Job processing failed, requeue.
		g.logger.Warningf("error processing %s job (requeued): %v", objKey, err)
		g.queue.AddRateLimited(objKey)
	} else {
		g.logger.Errorf("Error processing %s: %v", objKey, err)
		g.queue.Forget(objKey)
	}

	return false
}

// processJob is where the real processing logic of the item is.
func (g *generic) processJob(objKey string) error {
	defer g.queue.Done(objKey)

	// Get the object
	obj, exists, err := g.informer.GetIndexer().GetByKey(objKey)
	if err != nil {
		return err
	}

	// handle the object.
	if !exists { // Deleted resource from the cache.
		start := time.Now()
		if err := g.handler.Delete(objKey); err != nil {
			g.metrics.IncResourceDeleteEventProcessedError(g.handlerName)
			g.metrics.ObserveDurationResourceDeleteEventProcessedError(g.handlerName, start)
			return err
		}
		g.metrics.IncResourceDeleteEventProcessedSuccess(g.handlerName)
		g.metrics.ObserveDurationResourceDeleteEventProcessedSuccess(g.handlerName, start)
		return nil
	}

	start := time.Now()
	if err := g.handler.Add(obj.(runtime.Object)); err != nil {
		g.metrics.IncResourceAddEventProcessedError(g.handlerName)
		g.metrics.ObserveDurationResourceAddEventProcessedError(g.handlerName, start)
		return err
	}
	g.metrics.IncResourceAddEventProcessedSuccess(g.handlerName)
	g.metrics.ObserveDurationResourceAddEventProcessedSuccess(g.handlerName, start)
	return nil
}
