// +build trace

package golog

func (l *logger) Trace(arg interface{}) {
	l.Debug(arg)
}

func (l *logger) Tracef(fmt string, args ...interface{}) {
	l.Debugf(fmt, args...)
}
