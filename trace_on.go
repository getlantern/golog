// +build trace

package golog

import (
	"bufio"
	"io"
)

func LoggerFor(prefix string) Logger {
	l := &logger{
		prefix: prefix + ": ",
	}
	l.traceOut = l.newTraceWriter()
	return l
}

func (l *logger) Trace(arg interface{}) {
	l.Debug(arg)
}

func (l *logger) Tracef(fmt string, args ...interface{}) {
	l.Debugf(fmt, args...)
}

func (l *logger) newTraceWriter() io.Writer {
	pr, pw := io.Pipe()
	br := bufio.NewReader(pr)
	go func() {
		for {
			line, err := br.ReadString('\n')
			if err == nil {
				// Log the line (minus the trailing newline)
				l.Trace(line[:len(line)-1])
			}
		}
	}()
	return pw
}
