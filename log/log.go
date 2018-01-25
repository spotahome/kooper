package log

import (
	"github.com/golang/glog"
)

// Logger is the interface that the loggers used by the library will use.
type Logger interface {
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// Dummy logger doesn't log anything
var Dummy = &dummy{}

type dummy struct{}

func (d *dummy) Infof(format string, args ...interface{})    {}
func (d *dummy) Warningf(format string, args ...interface{}) {}
func (d *dummy) Errorf(format string, args ...interface{})   {}

// Glog is a wrapper for glog logger.
type Glog struct{}

func (g *Glog) Infof(format string, args ...interface{}) {
	glog.Infof(format, args...)
}
func (g *Glog) Warningf(format string, args ...interface{}) {
	glog.Warningf(format, args...)
}
func (g *Glog) Errorf(format string, args ...interface{}) {
	glog.Errorf(format, args...)
}
