// +build !trace

package golog

func (l *logger) Trace(arg interface{}) {
	// do nothing
}

func (l *logger) Tracef(fmt string, args ...interface{}) {
	// do nothing
}
