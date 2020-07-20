# Kooper [![Build Status][travis-image]][travis-url] [![Go Report Card][goreport-image]][goreport-url] [![GoDoc][godoc-image]][godoc-url]

Kooper is a Go library to create simple and flexible Kubernetes [controllers]/operators, in a fast, decoupled and easy way.

In other words, is a small alternative to big frameworks like [Kubebuilder] or [operator-framework].

**Library refactored (`v2`), for `v2` use `import "github.com/spotahome/kooper/v2"`**

## Features

- Easy usage and fast to get it working.
- Extensible (Kooper doesn't get in your way).
- Simple core concepts
  - `Retriever` + `Handler` is a `controller`
  - An `operator` is also a `controller`.
- Metrics (extensible with Prometheus already implementated).
- Ready for core Kubernetes resources (pods, ingress, deployments...) and CRDs.
- Optional leader election system for controllers.

## V0 vs V2

First of all, we used `v2` instead of `v[01]`, because it changes the library as a whole, theres no backwards compatibility,
`v0` is stable and used in production, although you eventually will want to update to `v2` becasuse `v0` will not be updated.

Import with:

```golang
import "github.com/spotahome/kooper/v2"
```

Regarding the changes... To know all of them check the changelog but mainly we simplified everything. The
most relevant changes you will need to be aware and could impact are:

- Before there were concepts like `operator` and `controller`, now only `controller` (this is at library level, you can continue creating controllers/operators).
- Before the CRD management was inside the library, now this should be managed outside Kooper.
  - You can use [this][kube-code-generator] to generate these manifests to register outside Kooper.
  - This is because controllers and CRDs have different lifecycles.
- Refactored Prometheus metrics to be more reliable, so you will need to change dashboards/alerts.
- `Delete` event removed because wasn't reliable (Check `Garbage-collection` section).

## Getting started

The simplest example that prints pods would be this:

```go
    // Create our retriever so the controller knows how to get/listen for pod events.
    retr := controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
        ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
            return k8scli.CoreV1().Pods("").List(options)
        },
        WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
            return k8scli.CoreV1().Pods("").Watch(options)
        },
    })

    // Our domain logic that will print all pod events.
    hand := controller.HandlerFunc(func(_ context.Context, obj runtime.Object) error {
        pod := obj.(*corev1.Pod)
        logger.Infof("Pod event: %s/%s", pod.Namespace, pod.Name)
        return nil
    })

    // Create the controller with custom configuration.
    cfg := &controller.Config{
        Name:      "example-controller",
        Handler:   hand,
        Retriever: retr,
        Logger:    logger,
    }
    ctrl, err := controller.New(cfg)
    if err != nil {
        return fmt.Errorf("could not create controller: %w", err)
    }

    // Start our controller.
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    ctrl.Run(ctx)
```

## Kubernetes version compatibility

Kooper at this moment uses as base `v1.17`. But [check the integration test in CI][ci] to know the supported versions.

## When should I use Kooper?

### Alternatives

What is the difference between kooper and alternatives like [Kubebuilder] or [operator-framework]?

Kooper embraces the Go philosophy of having small simple components/libs and use them as you wish in combination of others, instead of trying to solve every use case and imposing everything by sacriying flexbility and adding complexity.

As an example using the web applications world as reference: We could say that Kooper is more like Go HTTP router/libs, on the other side, Kubebuilder and operator-framework are like Ruby on rails/Django style frameworks.

For example Kubebuilder comes with:

- Admission webhooks.
- Folder/package structure conventions.
- CRD clients generation.
- RBAC manifest generation.
- Not pluggable/flexible usage of metrics, logging, HTTP/K8s API clients...
- ...

Kooper instead solves most of the core controller/operator problems but as a simple, small and flexible library, and let the other problems (like admission webhooks) be solved by other libraries specialiced on that. e.g

- Use whatever you want to create your CRD clients, maybe you don't have CRDs at all! (e.g [kube-code-generator]).
- You can setup your admission webhooks outside your controller by using other libraries like (e.g [Kubewebhook]).
- You can create your RBAC manifests as you wish and evolve while you develop your controller.
- Set you prefered logging system/style (comes with logrus implementation).
- Implement your prefered metrics backend (comes with Prometheus implementaion).
- Use your own Kubernetes clients (Kubernetes go library, implemented by your own for a special case...).
- ...

### Simplicty VS optimization

Kooper embraces simplicity over optimization, it favors small APIs, simplicity and easy to use/test methods. Some examples:

- Each Kooper controller is independent, don't share anything unless the user says explicitly (e.g. 2 controllers receive the same handler).
- Kooper uses a different resource/event cache internally for each controller (less bugs/corner cases but less optimized).
- Kooper handler receives the K8s resource, the responsibility of how this object is used is on the user.
- Multiresource controllers are made with independent controllers on the same app.

## More examples

On the [examples] folder you have different examples, like regular controllers, operators, metrics based, leader election, multiresource type controllers...

## Core concepts

Concept doesn't do a distinction between Operators and controllers, all are controllers, the difference of both is on what resources are retrieved.

A controller is based on 3 simple concepts:

### Retriever

The component that lists and watch the resources the controller will handle when there is a change. Kooper comes with some helpers to create fast retrievers:

- `Retriever`: The core retriever it needs to implement list (list objects), and watch, subscribe to object changes.
- `RetrieverFromListerWatcher`: Converts a Kubernetes ListerWatcher into a kooper Retriever.

The `Retriever` can be based on Kubernetes base resources (Pod, Deployment, Service...) or based on CRDs, theres no distinction.

The `Retriever` is an interface so you can use the middleware/wrapper/decorator pattern to extend (e.g add custom metrics).

### Handler

Kooper handles all the events on the same handler:

- `Handler`: The interface that knows how to handle kubernetes objects.
- `HandlerFunc`: A helper that gets a `Handler` from a function so you don't need to create a new type to define your `Handler`.

The `Handler` is an interface so you can use the middleware/wrapper/decorator pattern to extend (e.g add custom metrics).

### Controller

The controller is the component that uses the `Handler` and `Retriever` to start a feedback loop controller process:

- On the first start it will use `controller.Retriever.List` to get all the resources and pass them to the `controller.Handler`.
- Then it will call `controller.Handler` for every change done in the resources using the `controller.Retriever.Watcher`.
- At regular intervals (3 minute by default) it will call `controller.Handler` with all resources in case we have missed a `Watch` event.

## Other concepts

### Leader election

Check [Leader election](docs/leader-election.md).

### Garbage collection

Kooper only handles the events of resources that exist, these are triggered when the resources being watched are updated or created. There is no delete event, so in order to clean the resources you have 2 ways of doing these:

- If your controller creates as a side effect new Kubernetes resources you can use [owner references][owner-ref] on the created objects.
- If you want a more flexible clean up process (e.g clean from a database or a 3rd party service) you can use [finalizers], check the [pod-terminator-operator][finalizer-example] example.

### Multiresource or secondary resources

Sometimes we have controllers that work on a main or primary resource and we also want to handle the events of a secondary resource that is based on the first one. For example, a deployment controller that watches the pods (secondary) that belong to the deployment (primary) handled.

After using multiresource controllers/retrievers, we though that we don't need a multiresource controller, this is not necesary becase:

- Adds complexity.
- Adds corner cases, this translates in bugs, e.g
  - Internal object cache based on IDs of `{namespace}/{name}` scheme (ignoring types).
    - Receiving a deletion watch event of one type removes the other type object with the same name from the cache (service and deployment have same ns and same name).
    - The different resources that share name and ns, will be only process one of the types (sometimes is useful, others adds bugs and corner cases).
- An error on one of the retrieval types stops all the controller handling process and not only the one based on that type.
- Programatically speaking, you can reuse the `Handler` in multiple controllers.

The solution to these problems, is to embrace simplicity once again, and mainly is creating multiple controllers using the same `Handler`, each controller with a different `ListerWatcher`. The `Handler` API is easy enough to reuse it across multiple controllers, check an [example][multiresource-example]. Also, this comes with extra benefits:

- Different controller interval depending on the type (fast changing secondary objects can reconcile faster than the primary one, or viceversa).
- Wrap the controller handler with a middlewre only for a particular type.
- One of the type retrieval fails, the other type controller continues working (running in degradation mode).
- Flexibility, e.g leader election for the primary type, no leader election for the secondary type.
- Controller config has a handy flag to disable resync (`DisableResync`), sometimes this can be useful on secondary resources (only act on changes).

[travis-image]: https://travis-ci.org/spotahome/kooper.svg?branch=master
[travis-url]: https://travis-ci.org/spotahome/kooper
[goreport-image]: https://goreportcard.com/badge/github.com/spotahome/kooper
[goreport-url]: https://goreportcard.com/report/github.com/spotahome/kooper
[godoc-image]: https://pkg.go.dev/badge/github.com/spotahome/kooper/v2
[godoc-url]: https://pkg.go.dev/github.com/spotahome/kooper/v2
[examples]: examples/
[grafana-dashboard]: https://grafana.com/dashboards/7082
[controllers]: https://kubernetes.io/docs/concepts/architecture/controller/
[kubebuilder]: https://github.com/kubernetes-sigs/kubebuilder
[operator-framework]: https://github.com/operator-framework
[kubewebhook]: https://github.com/slok/kubewebhook
[kube-code-generator]: https://github.com/slok/kube-code-generator
[owner-ref]: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents
[finalizers]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers
[finalizer-example]: examples/pod-terminator-operator/operator/operator.go
[multiresource-example]: examples/multi-resource-controller
[ci]: https://github.com/spotahome/kooper/actions
