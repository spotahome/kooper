package leaderelection

import (
	"fmt"
	"os"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/spotahome/kooper/log"
)

const (
	defLeaseDuration = 15 * time.Second
	defRenewDeadline = 10 * time.Second
	defRetryPeriod   = 2 * time.Second
)

// Runner knows how to run using the leader election.
type Runner interface {
	// Run will run if the instance takes the lead. It's a blocking action.
	Run(func() error) error
}

// runner is the leader election default implementation.
type runner struct {
	key          string
	namespace    string
	k8scli       kubernetes.Interface
	resourceLock resourcelock.Interface
	logger       log.Logger
}

// New returns a new leader election service.
func New(key, namespace string, k8scli kubernetes.Interface, logger log.Logger) (Runner, error) {
	r := &runner{
		key:       key,
		namespace: namespace,
		k8scli:    k8scli,
		logger:    logger,
	}

	if err := r.validate(); err != nil {
		return nil, err
	}

	if err := r.initResourceLock(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *runner) validate() error {
	// Error if no namespace set.
	if r.namespace == "" {
		return fmt.Errorf("running in leader election mode requires the namespace running")
	}
	// Key required
	if r.key == "" {
		return fmt.Errorf("running in leader election mode requires a key for identification the different instances")
	}
	return nil
}

func (r *runner) initResourceLock() error {
	// Create the lock resource for the leader election.
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	id := hostname + "_" + string(uuid.NewUUID())

	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: r.key, Host: id})

	rl, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		r.namespace,
		r.key,
		r.k8scli.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		},
	)
	if err != nil {
		return fmt.Errorf("error creating lock: %v", err)
	}

	r.resourceLock = rl
	return nil

}

func (r *runner) Run(f func() error) error {
	errC := make(chan error, 1) // Channel to get the function returning error.

	// The function to execute when leader aquired.
	lef := func(stopC <-chan struct{}) {
		r.logger.Infof("lead acquire, starting...")
		errC <- f() // send the error to the leaderelection creator (leaderelection.Service)
		<-stopC
	}

	// Create the leader election configuration
	lec := leaderelection.LeaderElectionConfig{
		Lock:          r.resourceLock,
		LeaseDuration: defLeaseDuration,
		RenewDeadline: defRenewDeadline,
		RetryPeriod:   defRetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: lef,
			OnStoppedLeading: func() {
				errC <- fmt.Errorf("leader election lost")
			},
		},
	}

	// Create the leader elector.
	le, err := leaderelection.NewLeaderElector(lec)
	if err != nil {
		return fmt.Errorf("error creating leader election: %s", err)
	}

	// Execute!
	r.logger.Infof("running in leader election mode, waiting to acquire leadership...")

	le.Run()

	err = <-errC
	return err
}
