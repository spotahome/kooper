package operator

import (
	"github.com/spotahome/kooper/client/crd"
	podtermk8scli "github.com/spotahome/kooper/examples/pod-terminator-operator/client/k8s/clientset/versioned"
	"github.com/spotahome/kooper/examples/pod-terminator-operator/log"
	"github.com/spotahome/kooper/operator"
	"github.com/spotahome/kooper/operator/controller"
	"k8s.io/client-go/kubernetes"
)

// New returns pod terminator operator.
func New(cfg Config, podTermCli podtermk8scli.Interface, crdCli crd.Interface, kubeCli kubernetes.Interface, logger log.Logger) (operator.Operator, error) {

	// Create crd.
	ptCRD := newPodTermiantorCRD(podTermCli, crdCli, kubeCli)

	// Create handler.
	handler := newHandler(kubeCli, logger)

	// Create controller.
	ctrl := controller.NewDefaultGeneric(cfg.ResyncPeriod, handler, ptCRD, logger)

	// Assemble CRD and controller to create the operator.
	return operator.NewBasicOperator(ptCRD, ctrl, logger), nil
}
