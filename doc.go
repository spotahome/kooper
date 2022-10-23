// Package kooper is a Go library to create simple and flexible Kubernetes controllers/operators easily.
// Is as simple as this:
//
//	// Create our retriever so the controller knows how to get/listen for pod events.
//	retr := controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
//	    ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
//	        return k8scli.CoreV1().Pods("").List(options)
//	    },
//	    WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
//	        return k8scli.CoreV1().Pods("").Watch(options)
//	    },
//	})
//
//	// Our domain logic that will print all pod events.
//	hand := controller.HandlerFunc(func(_ context.Context, obj runtime.Object) error {
//	    pod := obj.(*corev1.Pod)
//	    logger.Infof("Pod event: %s/%s", pod.Namespace, pod.Name)
//	    return nil
//	})
//
//	// Create the controller with custom configuration.
//	cfg := &controller.Config{
//	    Name:      "example-controller",
//	    Handler:   hand,
//	    Retriever: retr,
//	    Logger:    logger,
//	}
//	ctrl, err := controller.New(cfg)
//	if err != nil {
//	    return fmt.Errorf("could not create controller: %w", err)
//	}
//
//	// Start our controller.
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	ctrl.Run(ctx)
package kooper
