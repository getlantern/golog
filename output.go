package golog

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/getlantern/hidden"
)

type outputFn func(prefix string, skipFrames int, printStack bool, severity string, arg interface{}, values map[string]interface{})

// Output is a log output that can optionally support structured logging
type Output interface {
	// Write debug messages
	Debug(prefix string, skipFrames int, printStack bool, severity string, arg interface{}, values map[string]interface{})

	// Write error messages
	Error(prefix string, skipFrames int, printStack bool, severity string, arg interface{}, values map[string]interface{})
}

// TextOutput creates an output that writes text to different io.Writers for errors and debug
func TextOutput(errorWriter io.Writer, debugWriter io.Writer) Output {
	return &textOutput{
		E:  errorWriter,
		D:  debugWriter,
		pc: make([]uintptr, 10),
	}
}

type textOutput struct {
	// E is the error writer
	E io.Writer
	// D is the debug writer
	D  io.Writer
	pc []uintptr
}

func (o *textOutput) Error(prefix string, skipFrames int, printStack bool, severity string, arg interface{}, values map[string]interface{}) {
	o.print(o.E, prefix, skipFrames, printStack, severity, arg, values)
}

func (o *textOutput) Debug(prefix string, skipFrames int, printStack bool, severity string, arg interface{}, values map[string]interface{}) {
	o.print(o.D, prefix, skipFrames, printStack, severity, arg, values)
}

func (o *textOutput) print(writer io.Writer, prefix string, skipFrames int, printStack bool, severity string, arg interface{}, values map[string]interface{}) {
	buf := bufferPool.Get()
	defer bufferPool.Put(buf)

	GetPrepender()(buf)
	linePrefix := o.linePrefix(prefix, skipFrames)
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
			printContext(buf, values)
			buf.WriteByte('\n')
		} else {
			mlp := ml.MultiLinePrinter()
			first := true
			for {
				writeHeader()
				more := mlp(buf)
				if first {
					printContext(buf, values)
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
	_, err := writer.Write(b)
	if err != nil {
		errorOnLogging(err)
	}
	if printStack {
		o.printStack(writer)
	}
}

// attaches the file and line number corresponding to
// the log message
func (o *textOutput) linePrefix(prefix string, skipFrames int) string {
	runtime.Callers(skipFrames, o.pc)
	funcForPc := runtime.FuncForPC(o.pc[0])
	file, line := funcForPc.FileLine(o.pc[0] - 1)
	return fmt.Sprintf("%s%s:%d ", prefix, filepath.Base(file), line)
}

func (o *textOutput) printStack(writer io.Writer) {
	var b []byte
	buf := bytes.NewBuffer(b)
	for _, pc := range o.pc {
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
	if _, err := buf.WriteTo(writer); err != nil {
		errorOnLogging(err)
	}
}
