// +build !trace

package golog

import "io/ioutil"

func LoggerFor(prefix string) Logger {
	return &logger{
		prefix:   prefix + ": ",
		traceOut: ioutil.Discard,
	}
}

func (l *logger) Trace(arg interface{}) {
	// do nothing
}

func (l *logger) Tracef(fmt string, args ...interface{}) {
	// do nothing
}
