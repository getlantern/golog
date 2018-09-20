// Package golog implements logging functions that log errors to stderr and
// debug messages to stdout. Trace logging is also supported.
// Trace logs go to stdout as well, but they are only written if the program
// is run with environment variable "TRACE=true".
// A stack dump will be printed after the message if "PRINT_STACK=true".
package golog

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/getlantern/errors"
	"github.com/getlantern/hidden"
	"github.com/getlantern/ops"
	"github.com/oxtoacart/bpool"
)

const (
	// ERROR is an error Severity
	ERROR = 500

	// FATAL is an error Severity
	FATAL = 600

	debugSkipFrames = 5
	errorSkipFrames = 2
)

var (
	outs atomic.Value

	bufferPool = bpool.NewBufferPool(200)

	onFatal atomic.Value
)

// Severity is a level of error (higher values are more severe)
type Severity int

func (s Severity) String() string {
	switch s {
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func init() {
	DefaultOnFatal()
}

// SetOutputs configures golog to use a streaming backend that writes to the given Writers.
func SetOutputs(errorOut io.Writer, debugOut io.Writer) {
	outs.Store(&outputs{
		ErrorOut: errorOut,
		DebugOut: debugOut,
	})
	setBaseLoggerBuilder(func(prefix string, traceOn bool, printStack bool) baseLogger {
		return &streamLogger{
			prefix:     prefix + ": ",
			traceOn:    traceOn,
			printStack: printStack,
		}
	})
}

// ResetOutputs resets golog to use a streaming backend that writes to os.Stderr and os.Stdout respectively
func ResetOutputs() {
	SetOutputs(os.Stderr, os.Stdout)
}

func getOutputs() *outputs {
	return outs.Load().(*outputs)
}

// OnFatal configures golog to call the given function on any FATAL error. By
// default, golog calls os.Exit(1) on any FATAL error.
func OnFatal(fn func(err error)) {
	onFatal.Store(fn)
}

// DefaultOnFatal enables the default behavior for OnFatal
func DefaultOnFatal() {
	onFatal.Store(func(err error) {
		os.Exit(1)
	})
}

type outputs struct {
	ErrorOut io.Writer
	DebugOut io.Writer
}

// MultiLine is an interface for arguments that support multi-line output.
type MultiLine interface {
	// MultiLinePrinter returns a function that can be used to print the
	// multi-line output. The returned function writes one line to the buffer and
	// returns true if there are more lines to write. This function does not need
	// to take care of trailing carriage returns, golog handles that
	// automatically.
	MultiLinePrinter() func(buf *bytes.Buffer) bool
}

type baseLogger interface {
	// Debug logs to stdout
	Debug(arg interface{})
	// Debugf logs to stdout
	Debugf(message string, args ...interface{})
	// Debugw logs with structured parameters from keysAndValues
	Debugw(message string, keysAndValues ...interface{})

	// Error logs to stderr
	Error(arg interface{}) error
	// Errorf logs to stderr. It returns the first argument that's an error, or
	// a new error built using fmt.Errorf if none of the arguments are errors.
	Errorf(message string, args ...interface{}) error
	// Errorw logs errors with structured parameters from keysAndValues
	Errorw(message string, keysAndValues ...interface{}) error

	// Fatal logs to stderr and then exits with status 1
	Fatal(arg interface{})
	// Fatalf logs to stderr and then exits with status 1
	Fatalf(message string, args ...interface{})
	// Fatalw logs errors with structured parameters from keysAndValues
	Fatalw(message string, keysAndValues ...interface{})

	// Trace logs to stderr only if TRACE=true
	Trace(arg interface{})
	// Tracef logs to stderr only if TRACE=true
	Tracef(message string, args ...interface{})
	// Tracew logs errors with structured parameters from keysAndValues
	Tracew(message string, keysAndValues ...interface{})

	// AsStdLogger returns a standard logger
	AsStdLogger() *log.Logger
}

type Logger interface {
	baseLogger

	// IsTraceEnabled() indicates whether or not tracing is enabled for this
	// logger.
	IsTraceEnabled() bool
}

// LoggerFor constructs a logger for the given prefix
func LoggerFor(prefix string) Logger {
	return &loggerFacade{
		prefix:     prefix,
		traceOn:    isTraceEnabled(prefix),
		printStack: isStackEnabled(),
	}
}

func isTraceEnabled(prefix string) bool {
	trace := os.Getenv("TRACE")
	traceOn, _ := strconv.ParseBool(trace)
	if traceOn {
		return true
	}
	prefixes := strings.Split(trace, ",")
	for _, p := range prefixes {
		if prefix == strings.Trim(p, " ") {
			return true
		}
	}
	return false
}

func isStackEnabled() bool {
	printStack, _ := strconv.ParseBool(os.Getenv("PRINT_STACK"))
	return printStack
}

type streamLogger struct {
	prefix     string
	traceOn    bool
	printStack bool
}

// attaches the file and line number corresponding to
// the log message
func (l *streamLogger) linePrefix(skipFrames int) (string, []uintptr) {
	pc := make([]uintptr, 10)
	runtime.Callers(skipFrames, pc)
	funcForPc := runtime.FuncForPC(pc[0])
	file, line := funcForPc.FileLine(pc[0] - 1)
	return fmt.Sprintf("%s%s:%d ", l.prefix, filepath.Base(file), line), pc
}

func (l *streamLogger) print(additionalContext []interface{}, out io.Writer, skipFrames int, severity string, arg interface{}) string {
	buf := bufferPool.Get()
	defer bufferPool.Put(buf)

	linePrefix, pc := l.linePrefix(skipFrames)
	writeHeader := func() {
		buf.WriteString(severity)
		buf.WriteString(" ")
		buf.WriteString(linePrefix)
	}
	if arg != nil {
		ml, isMultiline := arg.(MultiLine)
		if !isMultiline {
			writeHeader()
			fmt.Fprintf(buf, "%v", arg)
			printContext(additionalContext, buf, arg)
			buf.WriteByte('\n')
		} else {
			mlp := ml.MultiLinePrinter()
			first := true
			for {
				writeHeader()
				more := mlp(buf)
				if first {
					printContext(additionalContext, buf, arg)
					first = false
				}
				buf.WriteByte('\n')
				if !more {
					break
				}
			}
		}
	}
	b := []byte(hidden.Clean(buf.String()))
	_, err := out.Write(b)
	if err != nil {
		errorOnLogging(err)
	}
	if l.printStack {
		l.doPrintStack(pc)
	}

	return linePrefix
}

func (l *streamLogger) printf(additionalContext []interface{}, out io.Writer, skipFrames int, severity string, err error, message string, args ...interface{}) string {
	buf := bufferPool.Get()
	defer bufferPool.Put(buf)

	linePrefix, pc := l.linePrefix(skipFrames)
	buf.WriteString(severity)
	buf.WriteString(" ")
	buf.WriteString(linePrefix)
	fmt.Fprintf(buf, message, args...)
	printContext(additionalContext, buf, err)
	buf.WriteByte('\n')
	b := []byte(hidden.Clean(buf.String()))
	_, err2 := out.Write(b)
	if err2 != nil {
		errorOnLogging(err)
	}
	if l.printStack {
		l.doPrintStack(pc)
	}
	return linePrefix
}

func (l *streamLogger) Debug(arg interface{}) {
	l.print(nil, getOutputs().DebugOut, debugSkipFrames, "DEBUG", arg)
}

func (l *streamLogger) Debugf(message string, args ...interface{}) {
	l.printf(nil, getOutputs().DebugOut, debugSkipFrames, "DEBUG", nil, message, args...)
}

func (l *streamLogger) Debugw(message string, keyValuePairs ...interface{}) {
	l.print(keyValuePairs, getOutputs().DebugOut, debugSkipFrames, "DEBUG", message)
}

func (l *streamLogger) Error(arg interface{}) error {
	return l.errorSkipFrames(nil, arg, errorSkipFrames, ERROR)
}

func (l *streamLogger) Errorf(message string, args ...interface{}) error {
	return l.errorSkipFrames(nil, errors.NewOffset(errorSkipFrames, message, args...), errorSkipFrames, ERROR)
}

func (l *streamLogger) Errorw(message string, keyValuePairs ...interface{}) error {
	return l.errorSkipFrames(keyValuePairs, message, errorSkipFrames, ERROR)
}

func (l *streamLogger) Fatal(arg interface{}) {
	fatal(l.errorSkipFrames(nil, arg, errorSkipFrames, FATAL))
}

func (l *streamLogger) Fatalf(message string, args ...interface{}) {
	fatal(l.errorSkipFrames(nil, errors.NewOffset(errorSkipFrames, message, args...), errorSkipFrames, FATAL))
}

func (l *streamLogger) Fatalw(message string, keyValuePairs ...interface{}) {
	fatal(l.errorSkipFrames(keyValuePairs, message, errorSkipFrames, FATAL))
}

func fatal(err error) {
	fn := onFatal.Load().(func(err error))
	fn(err)
}

func (l *streamLogger) errorSkipFrames(additionalContext []interface{}, arg interface{}, skipFrames int, severity Severity) error {
	var err error
	switch e := arg.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}
	l.print(additionalContext, getOutputs().ErrorOut, skipFrames+4, severity.String(), err)
	return err
}

func (l *streamLogger) Trace(arg interface{}) {
	if l.traceOn {
		l.print(nil, getOutputs().DebugOut, debugSkipFrames, "TRACE", arg)
	}
}

func (l *streamLogger) Tracef(message string, args ...interface{}) {
	if l.traceOn {
		l.printf(nil, getOutputs().DebugOut, debugSkipFrames, "TRACE", nil, message, args...)
	}
}

func (l *streamLogger) Tracew(message string, keyValuePairs ...interface{}) {
	if l.traceOn {
		l.print(keyValuePairs, getOutputs().DebugOut, debugSkipFrames, "TRACE", message)
	}
}

type errorWriter struct {
	l *streamLogger
}

// Write implements method of io.Writer, due to different call depth,
// it will not log correct file and line prefix
func (w *errorWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	if s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	w.l.print(nil, getOutputs().ErrorOut, 6, "ERROR", s)
	return len(p), nil
}

func (l *streamLogger) AsStdLogger() *log.Logger {
	return log.New(&errorWriter{l}, "", 0)
}

func (l *streamLogger) doPrintStack(pc []uintptr) {
	var b []byte
	buf := bytes.NewBuffer(b)
	for _, pc := range pc {
		funcForPc := runtime.FuncForPC(pc)
		if funcForPc == nil {
			break
		}
		name := funcForPc.Name()
		if strings.HasPrefix(name, "runtime.") {
			break
		}
		file, line := funcForPc.FileLine(pc)
		fmt.Fprintf(buf, "\t%s\t%s: %d\n", name, file, line)
	}
	if _, err := buf.WriteTo(os.Stderr); err != nil {
		errorOnLogging(err)
	}
}

func errorOnLogging(err error) {
	fmt.Fprintf(os.Stderr, "Unable to log: %v\n", err)
}

func printContext(additionalContext []interface{}, buf *bytes.Buffer, err interface{}) {
	// Note - we don't include globals when printing in order to avoid polluting the text log
	values := ops.AsMap(err, false)
	if len(additionalContext) > 0 && len(additionalContext)%2 == 0 {
		if values == nil {
			values = make(map[string]interface{})
		}
		for i := 0; i < len(additionalContext); i += 2 {
			values[additionalContext[i].(string)] = additionalContext[i+1]
		}
	}
	if len(values) == 0 {
		return
	}
	buf.WriteString(" [")
	var keys []string
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		value := values[key]
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(key)
		buf.WriteString("=")
		fmt.Fprintf(buf, "%v", value)
	}
	buf.WriteByte(']')
}
