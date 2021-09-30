package operator

import (
	"github.com/yxxhero/kooper/client/crd"
	"github.com/yxxhero/kooper/operator"
	"github.com/yxxhero/kooper/operator/controller"
	"k8s.io/client-go/kubernetes"

	podtermk8scli "github.com/yxxhero/kooper/examples/pod-terminator-operator/client/k8s/clientset/versioned"
	"github.com/yxxhero/kooper/examples/pod-terminator-operator/log"
)

// New returns pod terminator operator.
func New(cfg Config, podTermCli podtermk8scli.Interface, crdCli crd.Interface, kubeCli kubernetes.Interface, logger log.Logger) (operator.Operator, error) {

	// Create crd.
	ptCRD := newPodTermiantorCRD(podTermCli, crdCli, kubeCli)

	// Create handler.
	handler := newHandler(kubeCli, logger)

	// Create controller.
	ctrl := controller.NewSequential(cfg.ResyncPeriod, handler, ptCRD, nil, logger)

	// Assemble CRD and controller to create the operator.
	return operator.NewOperator(ptCRD, ctrl, logger), nil
}
