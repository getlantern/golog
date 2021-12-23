package golog

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getlantern/errors"
	"github.com/getlantern/ops"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/stretchr/testify/assert"
)

var (
	expectedLog     = "SEVERITY myprefix: golog_test.go:999 Hello world\nSEVERITY myprefix: golog_test.go:999 Hello true [cvarA=a cvarB=b op=name root_op=name]\n"
	expectedLogJson = `{"level": "DEBUG", "component": "myprefix", "caller":"golog_test.go:999", "msg": "Hello world"}
{"level": "DEBUG", "component": "myprefix", "caller":"golog_test.go:999", "msg": "Hello true", "context": {"cvarA":"a", "cvarB":"b", "op":"name", "root_op":"name"}}`
	expectedErrorLogJson = `{"level": "ERROR", "component": "myprefix", "caller":"golog_test.go:999", "msg": "Hello world\n  at github.com/getlantern/golog.TestErrorJson (golog_test.go:999)\n  at testing.tRunner (testing.go:999)\n  at runtime.goexit (asm_amd999.s:999)\nCaused by: world\n  at github.com/getlantern/golog.errorReturner (golog_test.go:999)\n  at github.com/getlantern/golog.TestErrorJson (golog_test.go:999)\n  at testing.tRunner (testing.go:999)\n  at runtime.goexit (asm_amd999.s:999)\n", "context":{"cvarC":"c","cvarD":"d","error":"Hello %v","error_location":"github.com/getlantern/golog.TestErrorJson (golog_test.go:999)","error_text":"Hello world","error_type":"errors.Error","op":"name","root_op":"name"}}
{"level": "ERROR", "component": "myprefix", "caller":"golog_test.go:999", "msg": "Hello true\n  at github.com/getlantern/golog.TestErrorJson (golog_test.go:999)\n  at testing.tRunner (testing.go:999)\n  at runtime.goexit (asm_amd999.s:999)\nCaused by: Hello\n  at github.com/getlantern/golog.TestErrorJson (golog_test.go:999)\n  at testing.tRunner (testing.go:999)\n  at runtime.goexit (asm_amd999.s:999)\n", "context":{"cvarA":"a", "cvarB":"b", "cvarC":"c", "error":"%v %v", "error_location":"github.com/getlantern/golog.TestErrorJson (golog_test.go:999)", "error_text":"Hello true", "error_type":"errors.Error", "op":"name999", "root_op":"name999"}}
`
	expectedErrorLog = `ERROR myprefix: golog_test.go:999 Hello world [cvarC=c cvarD=d error=Hello %v error_location=github.com/getlantern/golog.TestError (golog_test.go:999) error_text=Hello world error_type=errors.Error op=name root_op=name]
ERROR myprefix: golog_test.go:999   at github.com/getlantern/golog.TestError (golog_test.go:999)
ERROR myprefix: golog_test.go:999   at testing.tRunner (testing.go:999)
ERROR myprefix: golog_test.go:999   at runtime.goexit (asm_amd999.s:999)
ERROR myprefix: golog_test.go:999 Caused by: world
ERROR myprefix: golog_test.go:999   at github.com/getlantern/golog.errorReturner (golog_test.go:999)
ERROR myprefix: golog_test.go:999   at github.com/getlantern/golog.TestError (golog_test.go:999)
ERROR myprefix: golog_test.go:999   at testing.tRunner (testing.go:999)
ERROR myprefix: golog_test.go:999   at runtime.goexit (asm_amd999.s:999)
ERROR myprefix: golog_test.go:999 Hello true [cvarA=a cvarB=b cvarC=c error=%v %v error_location=github.com/getlantern/golog.TestError (golog_test.go:999) error_text=Hello true error_type=errors.Error op=name999 root_op=name999]
ERROR myprefix: golog_test.go:999   at github.com/getlantern/golog.TestError (golog_test.go:999)
ERROR myprefix: golog_test.go:999   at testing.tRunner (testing.go:999)
ERROR myprefix: golog_test.go:999   at runtime.goexit (asm_amd999.s:999)
ERROR myprefix: golog_test.go:999 Caused by: Hello
ERROR myprefix: golog_test.go:999   at github.com/getlantern/golog.TestError (golog_test.go:999)
ERROR myprefix: golog_test.go:999   at testing.tRunner (testing.go:999)
ERROR myprefix: golog_test.go:999   at runtime.goexit (asm_amd999.s:999)
`
	expectedTraceLog = "TRACE myprefix: golog_test.go:999 Hello world\nTRACE myprefix: golog_test.go:999 Hello true\nTRACE myprefix: golog_test.go:999 Gravy\nTRACE myprefix: golog_test.go:999 TraceWriter closed due to unexpected error: EOF\n"
	expectedStdLog   = expectedLog
)

var (
	replaceNumbers = regexp.MustCompile("[0-9]+")
)

func init() {
	ops.SetGlobal("global", "shouldn't show up")
}

func expected(severity string, log string) string {
	return strings.Replace(log, "SEVERITY", severity, -1)
}

func normalized(log string) string {
	return replaceNumbers.ReplaceAllString(log, "999")
}

func TestReport(t *testing.T) {
	SetOutputs(ioutil.Discard, ioutil.Discard)
	OnFatal(func(err error) {
		// ignore (prevents test from exiting)
	})

	errorCount := 0
	fatalCount := 0
	RegisterReporter(func(err error, severity Severity, ctx map[string]interface{}) {
		switch severity {
		case ERROR:
			errorCount++
		case FATAL:
			fatalCount++
		}
	})
	l := LoggerFor("reporting")
	assert.Error(t, l.Error("Some error"))
	l.Fatal("Fatal error")
	assert.Equal(t, 1, errorCount)
	assert.Equal(t, 1, fatalCount)
}

func TestDebug(t *testing.T) {
	out := newBuffer()
	SetOutputs(ioutil.Discard, out)
	l := LoggerFor("myprefix")
	l.Debug("Hello world")
	defer ops.Begin("name").Set("cvarA", "a").Set("cvarB", "b").End()
	l.Debugf("Hello %v", true)
	assert.Equal(t, expected("DEBUG", expectedLog), out.String())
}

func TestDebugJson(t *testing.T) {
	out := newBuffer()
	SetOutput(JsonOutput(ioutil.Discard, out))
	l := LoggerFor("myprefix")
	l.Debug("Hello world")
	defer ops.Begin("name").Set("cvarA", "a").Set("cvarB", "b").End()
	l.Debugf("Hello %v", true)
	expectedLines := strings.Split(expectedLogJson, "\n")
	gotLines := strings.Split(strings.TrimSpace(out.String()), "\n")
	assert.Equal(t, len(expectedLines), len(gotLines))
	for i := range expectedLines {
		var expected Event
		var got Event
		assert.NoError(t, json.Unmarshal([]byte(expectedLines[i]), &expected))
		assert.NoError(t, json.Unmarshal([]byte(gotLines[i]), &got))
		assert.EqualValues(t, expected, got)
	}
}

func TestDebugZap(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.DebugLevel)
	zl := zap.New(observedZapCore)

	SetOutput(ZapOutput(zl))
	l := LoggerFor("myprefix")
	l.Debug("Hello world")
	defer ops.Begin("name").Set("cvarA", "a").Set("cvarB", "b").End()
	l.Debugf("Hello %v", true)
	entries := observedLogs.All()
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, "Hello world", entries[0].Message)
	assert.Equal(t, "name", entries[1].ContextMap()["op"])
}

func TestErrorZap(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.DebugLevel)
	zl := zap.New(observedZapCore, zap.AddStacktrace(zap.DebugLevel), zap.AddCaller())

	SetOutput(ZapOutput(zl))
	l := LoggerFor("myprefix")
	ctx := ops.Begin("name").Set("cvarC", "c")
	err := errorReturner()
	err1 := errors.New("Hello %v", err)
	err2 := errors.New("Hello")
	ctx.End()
	assert.Error(t, l.Error(err1))
	defer ops.Begin("name2").Set("cvarA", "a").Set("cvarB", "b").End()
	assert.Error(t, l.Errorf("%v %v", err2, true))
	entries := observedLogs.All()
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, "github.com/getlantern/golog.TestErrorZap", entries[0].Entry.Caller.Function)
}

func TestError(t *testing.T) {
	out := newBuffer()
	SetOutputs(out, ioutil.Discard)
	l := LoggerFor("myprefix")
	ctx := ops.Begin("name").Set("cvarC", "c")
	err := errorReturner()
	err1 := errors.New("Hello %v", err)
	err2 := errors.New("Hello")
	ctx.End()
	assert.Error(t, l.Error(err1))
	defer ops.Begin("name2").Set("cvarA", "a").Set("cvarB", "b").End()
	assert.Error(t, l.Errorf("%v %v", err2, true))
	t.Log(out.String())
	assert.Equal(t, expectedErrorLog, out.String())
}

func TestErrorJson(t *testing.T) {
	out := newBuffer()
	SetOutput(JsonOutput(out, ioutil.Discard))
	l := LoggerFor("myprefix")
	ctx := ops.Begin("name").Set("cvarC", "c")
	err := errorReturner()
	err1 := errors.New("Hello %v", err)
	err2 := errors.New("Hello")
	ctx.End()
	assert.Error(t, l.Error(err1))
	defer ops.Begin("name2").Set("cvarA", "a").Set("cvarB", "b").End()
	assert.Error(t, l.Errorf("%v %v", err2, true))
	expectedLines := strings.Split(strings.TrimSpace(expectedErrorLogJson), "\n")
	gotLines := strings.Split(strings.TrimSpace(out.String()), "\n")
	assert.Equal(t, len(expectedLines), len(gotLines))
	for i := range expectedLines {
		var expected Event
		var got Event
		assert.NoError(t, json.Unmarshal([]byte(expectedLines[i]), &expected))
		assert.NoError(t, json.Unmarshal([]byte(gotLines[i]), &got))
		assert.EqualValues(t, expected, got)
	}
}

func errorReturner() error {
	defer ops.Begin("name").Set("cvarD", "d").End()
	return errors.New("world")
}

func TestTraceEnabled(t *testing.T) {
	originalTrace := os.Getenv("TRACE")
	err := os.Setenv("TRACE", "true")
	if err != nil {
		t.Fatalf("Unable to set trace to true")
	}
	defer func() {
		if err := os.Setenv("TRACE", originalTrace); err != nil {
			t.Fatalf("Unable to set TRACE environment variable: %v", err)
		}
	}()

	out := newBuffer()
	SetOutputs(ioutil.Discard, out)
	l := LoggerFor("myprefix")
	l.Trace("Hello world")
	l.Tracef("Hello %v", true)
	tw := l.TraceOut()
	if _, err := tw.Write([]byte("Gravy\n")); err != nil {
		t.Fatalf("Unable to write: %v", err)
	}
	if err := tw.(io.Closer).Close(); err != nil {
		t.Fatalf("Unable to close: %v", err)
	}

	// Give trace writer a moment to catch up
	time.Sleep(50 * time.Millisecond)
	assert.Regexp(t, expected("TRACE", expectedTraceLog), out.String())
}

func TestTraceDisabled(t *testing.T) {
	originalTrace := os.Getenv("TRACE")
	err := os.Setenv("TRACE", "false")
	if err != nil {
		t.Fatalf("Unable to set trace to false")
	}
	defer func() {
		if err := os.Setenv("TRACE", originalTrace); err != nil {
			t.Fatalf("Unable to set TRACE environment variable: %v", err)
		}
	}()

	out := newBuffer()
	SetOutputs(ioutil.Discard, out)
	l := LoggerFor("myprefix")
	l.Trace("Hello world")
	l.Tracef("Hello %v", true)
	if _, err := l.TraceOut().Write([]byte("Gravy\n")); err != nil {
		t.Fatalf("Unable to write: %v", err)
	}

	// Give trace writer a moment to catch up
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, "", out.String(), "Nothing should have been logged")
}

func TestAsStdLogger(t *testing.T) {
	out := newBuffer()
	SetOutputs(out, ioutil.Discard)
	l := LoggerFor("myprefix")
	stdlog := l.AsStdLogger()
	stdlog.Print("Hello world")
	defer ops.Begin("name").Set("cvarA", "a").Set("cvarB", "b").End()
	stdlog.Printf("Hello %v", true)
	assert.Equal(t, expected("ERROR", expectedStdLog), out.String())
}

// TODO: TraceWriter appears to have been broken since we added line numbers
// func TestTraceWriter(t *testing.T) {
// 	originalTrace := os.Getenv("TRACE")
// 	err := os.Setenv("TRACE", "true")
// 	if err != nil {
// 		t.Fatalf("Unable to set trace to true")
// 	}
// 	defer func() {
// 		if err := os.Setenv("TRACE", originalTrace); err != nil {
// 			t.Fatalf("Unable to set TRACE environment variable: %v", err)
// 		}
// 	}()
//
// 	out := newBuffer()
// 	SetOutputs(ioutil.Discard, out)
// 	l := LoggerFor("myprefix")
// 	trace := l.TraceOut()
// 	trace.Write([]byte("Hello world\n"))
// 	defer ops.Begin().Set("cvarA", "a").Set("cvarB", "b").End()
// 	trace.Write([]byte("Hello true\n"))
// 	assert.Equal(t, expected("TRACE", expectedStdLog), out.String())
// }

func newBuffer() *synchronizedbuffer {
	return &synchronizedbuffer{orig: &bytes.Buffer{}}
}

type synchronizedbuffer struct {
	orig  *bytes.Buffer
	mutex sync.RWMutex
}

func (buf *synchronizedbuffer) Write(p []byte) (int, error) {
	buf.mutex.Lock()
	defer buf.mutex.Unlock()
	return buf.orig.Write(p)
}

func (buf *synchronizedbuffer) String() string {
	buf.mutex.RLock()
	defer buf.mutex.RUnlock()
	return normalized(buf.orig.String())
}
