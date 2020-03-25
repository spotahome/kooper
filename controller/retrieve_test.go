package controller_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/spotahome/kooper/controller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

var (
	testPodList = &corev1.PodList{
		Items: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "test1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "test2"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "test3"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "test4"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "test5"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "test6"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "test7"}},
		},
	}
	testEventList = []watch.Event{
		{Type: watch.Added, Object: &testPodList.Items[0]},
		{Type: watch.Added, Object: &testPodList.Items[1]},
		{Type: watch.Added, Object: &testPodList.Items[2]},
		{Type: watch.Added, Object: &testPodList.Items[3]},
		{Type: watch.Added, Object: &testPodList.Items[4]},
		{Type: watch.Added, Object: &testPodList.Items[5]},
		{Type: watch.Added, Object: &testPodList.Items[6]},
	}
)

func testPodListFunc(pl *corev1.PodList) cache.ListFunc {
	return func(options metav1.ListOptions) (runtime.Object, error) {
		return pl, nil
	}
}

func testEventWatchFunc(evs []watch.Event) cache.WatchFunc {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		cg := make(chan watch.Event)
		go func() {
			for _, ev := range evs {
				cg <- ev
			}
			close(cg)
		}()

		return watch.NewProxyWatcher(cg), nil
	}
}

func TestRetrieverFromListerWatcher(t *testing.T) {
	tests := map[string]struct {
		listerWatcher cache.ListerWatcher
		expList       runtime.Object
		expListErr    bool
		expWatch      []watch.Event
		expWatchErr   bool
	}{
		"A List error or a watch error should be propagated to the upper layer": {
			listerWatcher: &cache.ListWatch{
				ListFunc:  func(_ metav1.ListOptions) (runtime.Object, error) { return nil, fmt.Errorf("wanted error") },
				WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) { return nil, fmt.Errorf("wanted error") },
			},
			expListErr:  true,
			expWatchErr: true,
		},

		"List and watch should call the Kubernetes go clients lister watcher correctly.": {
			listerWatcher: &cache.ListWatch{
				ListFunc:  testPodListFunc(testPodList),
				WatchFunc: testEventWatchFunc(testEventList),
			},
			expList:  testPodList,
			expWatch: testEventList,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			ret := controller.MustRetrieverFromListerWatcher(test.listerWatcher)

			// Test list.
			objs, err := ret.List(context.TODO(), metav1.ListOptions{})
			if test.expListErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expList, objs)
			}

			// Test watch.
			w, err := ret.Watch(context.TODO(), metav1.ListOptions{})
			evs := []watch.Event{}
			if test.expWatchErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				for ev := range w.ResultChan() {
					evs = append(evs, ev)
				}
				assert.Equal(test.expWatch, evs)
			}
		})
	}
}

func TestMultiRetriever(t *testing.T) {
	tests := map[string]struct {
		retrievers  []controller.Retriever
		expList     func() runtime.Object
		expListErr  bool
		expWatch    func() []watch.Event
		expWatchErr bool
	}{
		"A List error or a watch error should be propagated to the upper layer if any of the retrievers fail.": {
			retrievers: []controller.Retriever{
				controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
					ListFunc:  testPodListFunc(testPodList),
					WatchFunc: testEventWatchFunc(testEventList),
				}),
				controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
					ListFunc:  func(_ metav1.ListOptions) (runtime.Object, error) { return nil, fmt.Errorf("wanted error") },
					WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) { return nil, fmt.Errorf("wanted error") },
				}),
				controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
					ListFunc:  testPodListFunc(testPodList),
					WatchFunc: testEventWatchFunc(testEventList),
				}),
			},
			expList:     func() runtime.Object { return nil },
			expListErr:  true,
			expWatch:    func() []watch.Event { return nil },
			expWatchErr: true,
		},

		"the lists and watch should be merged with the different retrievers result.": {
			retrievers: []controller.Retriever{
				controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
					ListFunc:  testPodListFunc(&corev1.PodList{Items: testPodList.Items[0:3]}),
					WatchFunc: testEventWatchFunc(testEventList[0:3]),
				}),
				controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
					ListFunc:  testPodListFunc(&corev1.PodList{Items: testPodList.Items[3:5]}),
					WatchFunc: testEventWatchFunc(testEventList[3:5]),
				}),
				controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
					ListFunc:  testPodListFunc(&corev1.PodList{Items: testPodList.Items[5:7]}),
					WatchFunc: testEventWatchFunc(testEventList[5:7]),
				}),
			},
			expList: func() runtime.Object {
				items, _ := meta.ExtractList(testPodList)
				l := &metav1.List{}
				for _, item := range items {
					l.Items = append(l.Items, runtime.RawExtension{Object: item})
				}

				return l
			},
			expWatch: func() []watch.Event {
				return testEventList
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			ret, err := controller.NewMultiRetriever(test.retrievers...)
			require.NoError(err)

			// Test list.
			objs, err := ret.List(context.TODO(), metav1.ListOptions{})
			if test.expListErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expList(), objs)
			}

			// Test watch.
			w, err := ret.Watch(context.TODO(), metav1.ListOptions{})
			evs := []watch.Event{}
			if test.expWatchErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				// Stop the watcher after some time so we can continue with the test.
				// We assume that we had enough time to get all the events.
				go func() {
					time.Sleep(20 * time.Millisecond)
					w.Stop()
				}()
				for ev := range w.ResultChan() {
					evs = append(evs, ev)
				}

				// Sort by object name.
				sort.SliceStable(evs, func(i, j int) bool {
					return evs[i].Object.(metav1.Object).GetName() < evs[j].Object.(metav1.Object).GetName()
				})

				assert.Equal(test.expWatch(), evs)
			}
		})
	}
}
