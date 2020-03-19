package controller

import (
	"context"

	"github.com/spotahome/kooper/controller"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/spotahome/kooper/examples/echo-pod-controller/log"
	"github.com/spotahome/kooper/examples/echo-pod-controller/service"
)

// New returns a new Echo controller.
func New(config Config, k8sCli kubernetes.Interface, logger log.Logger) (controller.Controller, error) {

	ret := NewPodRetrieve(config.Namespace, k8sCli)
	echoSrv := service.NewSimpleEcho(logger)
	handler := &handler{echoSrv: echoSrv}

	cfg := &controller.Config{
		Handler: handler,
		Retriever: ret,
		Logger: logger,

		ResyncInterval: config.ResyncPeriod,
	}

	return controller.New(cfg)
}

const (
	addPrefix    = "ADD"
	deletePrefix = "DELETE"
)

type handler struct {
	echoSrv service.Echo
}

func (h *handler) Add(_ context.Context, obj runtime.Object) error {
	h.echoSrv.EchoObj(addPrefix, obj)
	return nil
}
func (h *handler) Delete(_ context.Context, s string) error {
	h.echoSrv.EchoS(deletePrefix, s)
	return nil
}
