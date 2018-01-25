package handler

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// Handler knows how to handle the received resources from a kubernetes cluster.
type Handler interface {
	Add(obj runtime.Object) error
	Delete(string) error
}
