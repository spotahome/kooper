package handler

import (
	"context"
	"fmt"

	"github.com/spotahome/kooper/operator/common"
)

// Handler knows how to handle the received resources from a kubernetes cluster.
type Handler interface {
	Add(context.Context, *common.K8sEvent) error
	Delete(context.Context, *common.K8sEvent) error
}

// AddFunc knows how to handle resource adds.
type AddFunc func(context.Context, *common.K8sEvent) error

// DeleteFunc knows how to handle resource deletes.
type DeleteFunc func(context.Context, *common.K8sEvent) error

// HandlerFunc is a handler that is created from functions that the
// Handler interface requires.
type HandlerFunc struct {
	AddFunc    AddFunc
	DeleteFunc DeleteFunc
}

// Add satisfies Handler interface.
func (h *HandlerFunc) Add(ctx context.Context, evt *common.K8sEvent) error {
	if h.AddFunc == nil {
		return fmt.Errorf("function can't be nil")
	}
	return h.AddFunc(ctx, evt)
}

// Delete satisfies Handler interface.
func (h *HandlerFunc) Delete(ctx context.Context, evt *common.K8sEvent) error {
	if h.DeleteFunc == nil {
		return fmt.Errorf("function can't be nil")
	}
	return h.DeleteFunc(ctx, evt)
}
