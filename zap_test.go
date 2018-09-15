package golog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicLogging(t *testing.T) {
	log := zapLogger("tester")
	err := log.Error("test")

	assert.Error(t, err)

	err = log.Errorf("Error %v", "bop")

	assert.Error(t, err)

	std := log.AsStdLogger()

	std.Print("std test")

	log.Debug("test")
	log.Info("test")
	log.Debugf("test %v", "test")
	log.Infof("test %v", "test")
}
