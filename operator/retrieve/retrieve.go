package retrieve

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// Retriever is a way of wrapping  kubernetes lister watchers so they are easy to pass & manage them
// on Controllers.
type Retriever interface {
	GetListerWatcher() cache.ListerWatcher
	GetObject() runtime.Object
}
