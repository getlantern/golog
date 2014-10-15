// +build trace

package golog

func Trace(msg string) {
	Debug(msg)
}

func Tracef(fmt string, args ...interface{}) {
	Debugf(fmt, args...)
}
