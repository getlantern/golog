package golog

import (
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/getlantern/errors"
	"github.com/getlantern/ops"

	"github.com/stretchr/testify/assert"
)

var (
	expectedLog      = "SEVERITY myprefix: golog_test.go:999 Hello world\nSEVERITY myprefix: golog_test.go:999 Hello true [cvarA=a cvarB=b op=name root_op=name]\n"
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
	expectedDebugLog = "DEBUG myprefix: golog_test.go:999 Hello world\nDEBUG myprefix: golog_test.go:999 Hello true\n"
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

func TestInfo(t *testing.T) {
	out := newBuffer()
	SetOutputs(ioutil.Discard, out)
	l := NewLogger("myprefix")
	l.Info("Hello world")
	defer ops.Begin("name").Set("cvarA", "a").Set("cvarB", "b").End()
	l.Infof("Hello %v", true)
	assert.Equal(t, expected("INFO", expectedLog), out.String())
}

func TestError(t *testing.T) {
	out := newBuffer()
	SetOutputs(out, ioutil.Discard)
	l := NewLogger("myprefix")
	ctx := ops.Begin("name").Set("cvarC", "c")
	err := errorReturner()
	err1 := errors.New("Hello %v", err)
	err2 := errors.New("Hello")
	ctx.End()
	l.Error(err1)
	defer ops.Begin("name2").Set("cvarA", "a").Set("cvarB", "b").End()
	l.Errorf("%v %v", err2, true)
	t.Log(out.String())
	assert.Equal(t, expectedErrorLog, out.String())
}

func errorReturner() error {
	defer ops.Begin("name").Set("cvarD", "d").End()
	return errors.New("world")
}

func TestDebugEnabled(t *testing.T) {
	originalDebug := os.Getenv("DEBUG")
	err := os.Setenv("DEBUG", "true")
	if err != nil {
		t.Fatalf("Unable to set DEBUG to true")
	}
	defer func() {
		if err := os.Setenv("DEBUG", originalDebug); err != nil {
			t.Fatalf("Unable to set DEBUG environment variable: %v", err)
		}
	}()

	out := newBuffer()
	SetOutputs(ioutil.Discard, out)
	l := NewLogger("myprefix")
	l.Debug("Hello world")
	l.Debugf("Hello %v", true)
	assert.Regexp(t, expected("DEBUG", expectedDebugLog), out.String())
}

func TestDebugDisabled(t *testing.T) {
	originalDebug := os.Getenv("DEBUG")
	err := os.Setenv("DEBUG", "false")
	if err != nil {
		t.Fatalf("Unable to set DEBUG to false")
	}
	defer func() {
		if err := os.Setenv("DEBUG", originalDebug); err != nil {
			t.Fatalf("Unable to set DEBUG environment variable: %v", err)
		}
	}()

	out := newBuffer()
	SetOutputs(ioutil.Discard, out)
	l := NewLogger("myprefix")
	l.Debug("Hello world")
	l.Debugf("Hello %v", true)
	assert.Equal(t, "", out.String(), "Nothing should have been logged")
}

func TestAsStdLogger(t *testing.T) {
	out := newBuffer()
	SetOutputs(out, ioutil.Discard)
	l := NewLogger("myprefix")
	stdlog := l.AsStdLogger()
	stdlog.Print("Hello world")
	defer ops.Begin("name").Set("cvarA", "a").Set("cvarB", "b").End()
	stdlog.Printf("Hello %v", true)
	assert.Equal(t, expected("ERROR", expectedStdLog), out.String())
}

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
