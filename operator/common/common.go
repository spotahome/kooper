package common

import "k8s.io/apimachinery/pkg/runtime"

// K8sEvent represents the k8s events coming from the informer
type K8sEvent struct {
	Key       string
	HasSynced bool
	Object    runtime.Object
	Kind      string
}
