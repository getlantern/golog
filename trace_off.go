// +build !trace

package golog

func Trace(msg string) {
	// do nothing
}

func Tracef(fmt string, args ...interface{}) {
	// do nothing
}
