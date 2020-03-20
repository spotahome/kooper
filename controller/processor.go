package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/spotahome/kooper/monitoring/metrics"
)

// processor knows how to process object keys.
type processor interface {
	Process(ctx context.Context, key string) error
}

func newIndexerProcessor(indexer cache.Indexer, handler Handler) processor {
	return indexerProcessor{
		indexer: indexer,
		handler: handler,
	}
}

// indexerProcessor processes a key that will get the kubernetes object from a cache
// called indexer were the kubernetes watch updates have been indexed and stored by the
// listerwatchers from the informers.
type indexerProcessor struct {
	indexer cache.Indexer
	handler Handler
}

func (i indexerProcessor) Process(ctx context.Context, key string) error {
	// Get the object
	obj, exists, err := i.indexer.GetByKey(key)
	if err != nil {
		return err
	}

	// handle the object.
	if !exists { // Deleted resource from the cache.
		return i.handler.Delete(ctx, key)
	}

	return i.handler.Add(ctx, obj.(runtime.Object))
}

var errRequeued = fmt.Errorf("requeued after receiving error")

// retryProcessor will delegate the processing of a key to the received processor,
// in case the processing/handling of this key fails it will add the key
// again to a queue if it has retrys pending.
//
// If the processing errored and has been retried, it will return a `errRequeued`
// error.
type retryProcessor struct {
	name       string
	maxRetries int
	mrec       metrics.Recorder
	queue      workqueue.RateLimitingInterface
	next       processor
}

func newRetryProcessor(name string, maxRetries int, mrec metrics.Recorder, queue workqueue.RateLimitingInterface, next processor) processor {
	return retryProcessor{
		name:       name,
		maxRetries: maxRetries,
		mrec:       mrec,
		queue:      queue,
		next:       next,
	}
}

func (r retryProcessor) Process(ctx context.Context, key string) error {
	err := r.next.Process(ctx, key)

	// If there was an error and we have retries pending then requeue.
	if err != nil && r.queue.NumRequeues(key) < r.maxRetries {
		r.queue.AddRateLimited(key)
		r.mrec.IncResourceEventQueued(r.name, metrics.RequeueEvent)
		return fmt.Errorf("%w: %s", errRequeued, err)
	}

	r.queue.Forget(key)
	return err
}
