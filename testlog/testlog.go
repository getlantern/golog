package testlog

import (
	"testing"

	"github.com/getlantern/golog"
)

// Capture captures logs to the given testing.T's Log function.
// Returns a function that stops capturing logs.
//
// Typical usage:
//
//    func MyTest(t *testing.T) {
//        stopCapture := testlog.Capture(t)
//        defer stopCapture()
//        // do stuff
//    }
//
func Capture(t *testing.T) func() {
	w := &testLogWriter{t}
	return golog.SetOutputs(w, w)
}

type testLogWriter struct {
	*testing.T
}

func (w testLogWriter) Write(p []byte) (n int, err error) {
	w.Log((string)(p))
	return len(p), nil
}
