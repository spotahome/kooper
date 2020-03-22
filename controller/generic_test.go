package controller_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"github.com/spotahome/kooper/controller"
	"github.com/spotahome/kooper/controller/leaderelection"
	"github.com/spotahome/kooper/log"
	mcontroller "github.com/spotahome/kooper/mocks/controller"
)

// Namespace knows how to retrieve namespaces.
type namespaceRetriever struct {
	lw  cache.ListerWatcher
	obj runtime.Object
}

// NewNamespace returns a Namespace retriever.
func newNamespaceRetriever(client kubernetes.Interface) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1().Namespaces().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.CoreV1().Namespaces().Watch(options)
		},
	})
}

// GetListerWatcher knows how to retrieve Namespaces.
func (n *namespaceRetriever) GetListerWatcher() cache.ListerWatcher {
	return n.lw
}

// GetObject returns the namespace Object.
func (n *namespaceRetriever) GetObject() runtime.Object {
	return n.obj
}

func onKubeClientWatchNamespaceReturn(client *fake.Clientset, adds []*corev1.Namespace, updates []*corev1.Namespace, deletes []*corev1.Namespace) {
	w := watch.NewFake()
	client.AddWatchReactor("namespaces", func(action kubetesting.Action) (bool, watch.Interface, error) {
		return true, w, nil
	})

	go func() {
		// Adds.
		for _, obj := range adds {
			w.Add(obj)
		}
		// Updates.
		for _, obj := range updates {
			w.Modify(obj)
		}
		// Deletes.
		for _, obj := range deletes {
			w.Delete(obj)
		}
	}()
}

func onKubeClientListNamespaceReturn(client *fake.Clientset, nss *corev1.NamespaceList) {
	client.AddReactor("list", "namespaces", func(action kubetesting.Action) (bool, runtime.Object, error) {
		return true, nss, nil
	})
}

func createNamespaceList(prefix string, q int) (*corev1.NamespaceList, []*corev1.Namespace) {
	nss := []*corev1.Namespace{}
	nsl := &corev1.NamespaceList{
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
		Items: []corev1.Namespace{},
	}

	for i := 0; i < q; i++ {
		nsName := fmt.Sprintf("%s-%d", prefix, i)
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:            nsName,
				ResourceVersion: fmt.Sprintf("%d", i),
			},
		}

		nsl.Items = append(nsl.Items, ns)
		nss = append(nss, &ns)
	}

	return nsl, nss
}

func TestGenericControllerHandleAdds(t *testing.T) {
	nsList, expNSAdds := createNamespaceList("testing", 10)

	tests := []struct {
		name      string
		nsList    *corev1.NamespaceList
		expNSAdds []*corev1.Namespace
	}{
		{
			name:      "Listing multiple namespaces should call as add handlers for every namespace on list.",
			nsList:    nsList,
			expNSAdds: expNSAdds,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			controllerStopperC := make(chan struct{})
			resultC := make(chan error)

			// Mocks kubernetes  client.
			mc := &fake.Clientset{}
			onKubeClientListNamespaceReturn(mc, test.nsList)

			// Mock our handler and set expects.
			callHandling := 0 // used to track the number of calls.
			mh := &mcontroller.Handler{}
			for _, ns := range test.expNSAdds {
				mh.On("Add", mock.Anything, ns).Once().Return(nil).Run(func(args mock.Arguments) {
					callHandling++
					// Check last call, if is the last call expected then stop the controller so
					// we can assert the expectations of the calls and finish the test.
					if callHandling == len(test.expNSAdds) {
						close(controllerStopperC)
					}
				})
			}

			c, err := controller.New(&controller.Config{
				Name:      "test",
				Handler:   mh,
				Retriever: newNamespaceRetriever(mc),
				Logger:    log.Dummy,
			})
			require.NoError(err)

			// Run Controller in background.
			go func() {
				resultC <- c.Run(controllerStopperC)
			}()

			// Wait for different results. If no result means error failure.
			select {
			case err := <-resultC:
				if assert.NoError(err) {
					// Check handles from the controller.
					mh.AssertExpectations(t)
				}
			case <-time.After(1 * time.Second):
				assert.Fail("timeout waiting for controller handling, this could mean the controller is not receiving resources")

			}
		})
	}
}

func TestGenericControllerHandleDeletes(t *testing.T) {

	startNSList, expNSAdds := createNamespaceList("testing", 10)
	nsDels := []*corev1.Namespace{expNSAdds[0], expNSAdds[4], expNSAdds[1]}

	tests := []struct {
		name        string
		startNSList *corev1.NamespaceList
		deleteNs    []*corev1.Namespace
		expDeleteNs []*corev1.Namespace
	}{
		{
			name:        "Deleting multiple namespaces should call as delete handlers for every namespace on deleted.",
			startNSList: startNSList,
			deleteNs:    nsDels,
			expDeleteNs: nsDels,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			controllerStopperC := make(chan struct{})
			resultC := make(chan error)

			// Mocks kubernetes  client.
			mc := &fake.Clientset{}
			// Populate cache so we ensure deletes are correctly delivered.
			onKubeClientListNamespaceReturn(mc, test.startNSList)
			onKubeClientWatchNamespaceReturn(mc, nil, nil, test.deleteNs)

			// Mock our handler and set expects.
			callHandling := 0 // used to track the number of calls.
			mh := &mcontroller.Handler{}
			mh.On("Add", mock.Anything, mock.Anything).Return(nil)
			for _, ns := range test.expDeleteNs {
				mh.On("Delete", mock.Anything, ns.ObjectMeta.Name).Once().Return(nil).Run(func(args mock.Arguments) {
					// Check last call, if is the last call expected then stop the controller so
					// we can assert the expectations of the calls and finish the test.
					callHandling++
					if callHandling == len(test.expDeleteNs) {
						close(controllerStopperC)
					}
				})
			}

			c, err := controller.New(&controller.Config{
				Name:      "test",
				Handler:   mh,
				Retriever: newNamespaceRetriever(mc),
				Logger:    log.Dummy,
			})
			require.NoError(err)

			// Run Controller in background.
			go func() {
				resultC <- c.Run(controllerStopperC)
			}()

			// Wait for different results. If no result means error failure.
			select {
			case err := <-resultC:
				if assert.NoError(err) {
					// Check handles from the controller.
					mh.AssertExpectations(t)
				}
			case <-time.After(1 * time.Second):
				assert.Fail("timeout waiting for controller handling, this could mean the controller is not receiving resources")
			}
		})
	}
}

func TestGenericControllerErrorRetries(t *testing.T) {
	nsList, _ := createNamespaceList("testing", 11)

	tests := []struct {
		name        string
		nsList      *corev1.NamespaceList
		retryNumber int
	}{
		{
			name:        "Retrying N resources with M retries and error on all should be 1 + M processing calls per resource (N+N*M event processing calls).",
			nsList:      nsList,
			retryNumber: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			controllerStopperC := make(chan struct{})
			resultC := make(chan error)

			// Mocks kubernetes  client.
			mc := &fake.Clientset{}
			// Populate cache so we ensure deletes are correctly delivered.
			onKubeClientListNamespaceReturn(mc, nsList)

			// Mock our handler and set expects.
			totalCalls := len(test.nsList.Items) + len(test.nsList.Items)*test.retryNumber
			mh := &mcontroller.Handler{}
			err := fmt.Errorf("wanted error")

			// Expect all the retries
			for range test.nsList.Items {
				callsPerNS := test.retryNumber + 1 // initial call + retries.
				mh.On("Add", mock.Anything, mock.Anything).Return(err).Times(callsPerNS).Run(func(args mock.Arguments) {
					totalCalls--
					// Check last call, if is the last call expected then stop the controller so
					// we can assert the expectations of the calls and finish the test.
					if totalCalls <= 0 {
						close(controllerStopperC)
					}
				})
			}

			c, err := controller.New(&controller.Config{
				Name:                 "test",
				Handler:              mh,
				Retriever:            newNamespaceRetriever(mc),
				ProcessingJobRetries: test.retryNumber,
				Logger:               log.Dummy,
			})
			require.NoError(err)

			// Run Controller in background.
			go func() {
				resultC <- c.Run(controllerStopperC)
			}()

			// Wait for different results. If no result means error failure.
			select {
			case err := <-resultC:
				if assert.NoError(err) {
					// Check handles from the controller.
					mh.AssertExpectations(t)
				}
			case <-time.After(1 * time.Second):
				assert.Fail("timeout waiting for controller handling, this could mean the controller is not receiving resources")
			}
		})
	}
}

func TestGenericControllerWithLeaderElection(t *testing.T) {
	nsList, _ := createNamespaceList("testing", 5)

	tests := []struct {
		name        string
		nsList      *corev1.NamespaceList
		retryNumber int
	}{
		{
			name:   "",
			nsList: nsList,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			controllerStopperC := make(chan struct{})
			resultC := make(chan error)

			// Mocks kubernetes  client.
			mc := fake.NewSimpleClientset(nsList)

			// Mock our handler and set expects.
			mh1 := &mcontroller.Handler{}
			mh2 := &mcontroller.Handler{}
			mh3 := &mcontroller.Handler{}

			// Expect the calls on the lead (mh1) and no calls on the other ones.
			totalCalls := len(test.nsList.Items)
			mh1.On("Add", mock.Anything, mock.Anything).Return(nil).Times(totalCalls).Run(func(args mock.Arguments) {
				totalCalls--
				// Check last call, if is the last call expected then stop the controller so
				// we can assert the expectations of the calls and finish the test.
				if totalCalls <= 0 {
					close(controllerStopperC)
				}
			})

			nsret := newNamespaceRetriever(mc)

			// Leader election service.
			rlCfg := &leaderelection.LockConfig{
				LeaseDuration: 9999 * time.Second,
				RenewDeadline: 9998 * time.Second,
				RetryPeriod:   500 * time.Second,
			}
			lesvc1, _ := leaderelection.New("test", "default", rlCfg, mc, log.Dummy)
			lesvc2, _ := leaderelection.New("test", "default", rlCfg, mc, log.Dummy)
			lesvc3, _ := leaderelection.New("test", "default", rlCfg, mc, log.Dummy)

			c1, err := controller.New(&controller.Config{
				Name:                 "test1",
				Handler:              mh1,
				Retriever:            nsret,
				LeaderElector:        lesvc1,
				ProcessingJobRetries: test.retryNumber,
				Logger:               log.Dummy,
			})
			require.NoError(err)

			c2, err := controller.New(&controller.Config{
				Name:                 "test2",
				Handler:              mh2,
				Retriever:            nsret,
				LeaderElector:        lesvc2,
				ProcessingJobRetries: test.retryNumber,
				Logger:               log.Dummy,
			})
			require.NoError(err)

			c3, err := controller.New(&controller.Config{
				Name:                 "test3",
				Handler:              mh3,
				Retriever:            nsret,
				LeaderElector:        lesvc3,
				ProcessingJobRetries: test.retryNumber,
				Logger:               log.Dummy,
			})
			require.NoError(err)

			// Run multiple controller in background.
			go func() { resultC <- c1.Run(controllerStopperC) }()
			// Let the first controller became the leader.
			time.Sleep(200 * time.Microsecond)
			go func() { resultC <- c2.Run(controllerStopperC) }()
			go func() { resultC <- c3.Run(controllerStopperC) }()

			// Wait for different results. If no result means error failure.
			select {
			case err := <-resultC:
				if assert.NoError(err) {
					// Check handles from the controller.
					mh1.AssertExpectations(t)
					mh2.AssertExpectations(t)
					mh3.AssertExpectations(t)
				}
			case <-time.After(1 * time.Second):
				assert.Fail("timeout waiting for controller handling, this could mean the controller is not receiving resources")
			}
		})
	}
}
