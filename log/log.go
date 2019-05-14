package log

import (
	"fmt"
	"log"

	"k8s.io/klog"
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

// Klog is a wrapper for klog logger.
type Klog struct{}

// NewKlogger initialises the flags for klog
// If you use klog yourself, do not use this
// function. Instead, call klog.InitFlags
// yourself and create the Klog struct
func NewKlogger() *Klog {
	klog.InitFlags(nil)
	return &Klog{}
}

func (k *Klog) Infof(format string, args ...interface{}) {
	klog.Infof(format, args...)
}
func (k *Klog) Warningf(format string, args ...interface{}) {
	klog.Warningf(format, args...)
}
func (k *Klog) Errorf(format string, args ...interface{}) {
	klog.Errorf(format, args...)
}

// Std is a wrapper for go standard library logger.
type Std struct{}

func (s *Std) logWithPrefix(prefix, format string, args ...interface{}) {
	format = fmt.Sprintf("%s %s", prefix, format)
	log.Printf(format, args...)
}

func (s *Std) Infof(format string, args ...interface{}) {
	s.logWithPrefix("[INFO]", format, args...)
}
func (s *Std) Warningf(format string, args ...interface{}) {
	s.logWithPrefix("[WARN]", format, args...)
}
func (s *Std) Errorf(format string, args ...interface{}) {
	s.logWithPrefix("[ERROR]", format, args...)
}
