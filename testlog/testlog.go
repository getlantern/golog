package testlog

import (
	"errors"
	"sync"
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
	w := &testLogWriter{T: t, ch: make(chan []byte)}
	go w.run()
	reset := golog.SetOutputs(w, w)
	return func() {
		reset()
		w.stop()
	}
}

type testLogWriter struct {
	*testing.T
	mu      sync.Mutex
	stopped bool
	ch      chan []byte
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.ch <- p
		return len(p), nil
	}
	return 0, errors.New("writing to stopped testlog writer")
}

func (w *testLogWriter) stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopped = true
	close(w.ch)
}

func (w *testLogWriter) run() {
	for p := range w.ch {
		w.Log((string)(p))
	}
}
