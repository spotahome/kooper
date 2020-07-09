package logrus

import (
	"github.com/sirupsen/logrus"

	"github.com/spotahome/kooper/v2/log"
)

type logger struct {
	*logrus.Entry
}

// New returns a new log.Logger for a logrus implementation.
func New(l *logrus.Entry) log.Logger {
	return logger{Entry: l}
}

func (l logger) WithKV(kv log.KV) log.Logger {
	newLogger := l.Entry.WithFields(logrus.Fields(kv))
	return New(newLogger)
}
