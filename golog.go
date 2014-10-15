// package golog implements logging functions that log errors to stderr and
// debug messages to stdout. Trace logging is also supported. Trace logs go to
// stdout as well, but they are only written if the program is built with trace
// enabled, i.e.
//
//   -tags trace
//
package golog

import (
	"fmt"
	"os"
)

type Logger interface {
	// Debug logs to stdout
	Debug(arg interface{})
	// Debugf logs to stdout
	Debugf(message string, args ...interface{})

	// Error logs to stderr
	Error(arg interface{})
	// Errorf logs to stderr
	Errorf(message string, args ...interface{})

	// Trace logs to stderr only if -tags trace was specified at compile time
	Trace(arg interface{})
	// Tracef logs to stderr only if -tags trace was specified at compile time
	Tracef(message string, args ...interface{})

	// Fatal logs to stderr and then exits with status 1
	Fatal(arg interface{})
	// Fatalf logs to stderr and then exits with status 1
	Fatalf(message string, args ...interface{})
}

type logger struct {
	prefix string
}

func LoggerFor(prefix string) Logger {
	return &logger{prefix + ": "}
}

func (l *logger) Debug(arg interface{}) {
	fmt.Fprintf(os.Stdout, l.prefix+"%s", arg)
}

func (l *logger) Debugf(message string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, l.prefix+message+"\n", args...)
}

func (l *logger) Error(arg interface{}) {
	fmt.Fprintf(os.Stderr, l.prefix+"%s", arg)
}

func (l *logger) Errorf(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, l.prefix+message+"\n", args...)
}

func (l *logger) Fatal(arg interface{}) {
	l.Error(arg)
	os.Exit(1)
}

func (l *logger) Fatalf(message string, args ...interface{}) {
	l.Errorf(message, args...)
	os.Exit(1)
}
