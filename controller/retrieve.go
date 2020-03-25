package controller

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// Retriever is how a controller will retrieve the events on the resources from
// the APÎ server.
//
// A Retriever is bound to a single type.
type Retriever interface {
	List(ctx context.Context, options metav1.ListOptions) (runtime.Object, error)
	Watch(ctx context.Context, options metav1.ListOptions) (watch.Interface, error)
}

type listerWatcherRetriever struct {
	lw cache.ListerWatcher
}

// RetrieverFromListerWatcher returns a Retriever from a Kubernetes client-go cache.ListerWatcher.
// If the received lister watcher is nil it will error.
func RetrieverFromListerWatcher(lw cache.ListerWatcher) (Retriever, error) {
	if lw == nil {
		return nil, fmt.Errorf("listerWatcher can't be nil")
	}
	return listerWatcherRetriever{lw: lw}, nil
}

// MustRetrieverFromListerWatcher returns a Retriever from a Kubernetes client-go cache.ListerWatcher
// if there is an error it will panic.
func MustRetrieverFromListerWatcher(lw cache.ListerWatcher) Retriever {
	r, err := RetrieverFromListerWatcher(lw)
	if lw == nil {
		panic(err)
	}
	return r
}

func (l listerWatcherRetriever) List(_ context.Context, options metav1.ListOptions) (runtime.Object, error) {
	return l.lw.List(options)
}
func (l listerWatcherRetriever) Watch(_ context.Context, options metav1.ListOptions) (watch.Interface, error) {
	return l.lw.Watch(options)
}

type multiRetriever struct {
	rts []Retriever
}

// NewMultiRetriever returns a lister watcher that will list multiple types
//
// With this multi lister watcher a controller can receive updates in multiple types
// for example on pods and a deployments.
func NewMultiRetriever(retrievers ...Retriever) (Retriever, error) {
	for _, r := range retrievers {
		if r == nil {
			return nil, fmt.Errorf("at least one of the retrievers is nil")
		}
	}

	return multiRetriever{
		rts: retrievers,
	}, nil
}

func (m multiRetriever) List(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
	ls := &metav1.List{}
	for _, r := range m.rts {
		lo, err := r.List(ctx, *options.DeepCopy())
		if err != nil {
			return nil, err
		}

		items, err := meta.ExtractList(lo)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			ls.Items = append(ls.Items, runtime.RawExtension{Object: item})
		}
	}

	return ls, nil
}

func (m multiRetriever) Watch(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
	ws := make([]watch.Interface, len(m.rts))
	for i, rt := range m.rts {
		w, err := rt.Watch(ctx, options)
		if err != nil {
			return nil, err
		}
		ws[i] = w
	}

	return newMultiWatcher(ws...), nil
}

type multiWatcher struct {
	stopped bool
	mu      sync.Mutex
	stop    chan struct{}
	ch      chan watch.Event
	ws      []watch.Interface
}

func newMultiWatcher(ws ...watch.Interface) watch.Interface {
	m := &multiWatcher{
		stop: make(chan struct{}),
		ch:   make(chan watch.Event),
		ws:   ws,
	}

	// Run all watchers.
	// TODO(slok): call run here or lazy on ResultChan(), this last option can be dangerous (multiple calls).
	for _, w := range ws {
		go m.run(w)
	}

	return m
}

func (m *multiWatcher) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return
	}

	for _, w := range m.ws {
		w.Stop()
	}

	close(m.stop)
	close(m.ch)
	m.stopped = true
}

func (m *multiWatcher) ResultChan() <-chan watch.Event {
	return m.ch
}

func (m *multiWatcher) run(w watch.Interface) {
	c := w.ResultChan()
	for {
		select {
		case <-m.stop:
			return
		case e, ok := <-c:
			// Channel has been closed no need this loop anymore.
			if !ok {
				return
			}
			m.ch <- e
		}
	}
}
