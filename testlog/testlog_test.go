package testlog

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/getlantern/golog"
	"github.com/stretchr/testify/assert"
)

const (
	expectedCapture = `ERROR mytest: testlog_test.go:29 error 1
DEBUG mytest: buffer.go:54 debug 1
`
)

var (
	log = golog.LoggerFor("mytest")
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestTestLog(t *testing.T) {
	buf := &bytes.Buffer{}
	golog.SetOutputs(buf, buf)
	log.Error("error 1")
	stop := Capture(t)
	log.Error("this should show in test log")
	log.Debug("this should also show in test log")
	stop()
	log.Debug("debug 1")
	assert.Equal(t, expectedCapture, string(buf.Bytes()))
}

func TestConcurrent(t *testing.T) {
	// Note: Run "go test -count 10 -run Concurrent -race"
	golog.SetOutputs(ioutil.Discard, ioutil.Discard)
	stop := Capture(t)
	go func() {
		log.Debug("something")
	}()
	log.Debug("something")
	stop()
}
