package controller_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/spotahome/kooper/v2/controller"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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
