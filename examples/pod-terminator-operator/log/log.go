package log

import (
	"github.com/spotahome/kooper/v2/log"
)

// Logger is the interface of the operator logger. This is an example
// so our Loggger will be the same as the kooper one.
type Logger interface {
	log.Logger
}
