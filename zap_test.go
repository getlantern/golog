package golog

import (
	"testing"

	"go.uber.org/zap"
)

func TestBasicLogging(t *testing.T) {
	SetZapConfig(zap.NewDevelopmentConfig())

	log := ZapLogger("tester")
	log.Error("test")

	log.Errorf("Error %v", "bop")

	log.Debug("test")
	log.Info("test")
	log.Debugf("test %v", "test")
	log.Infof("test %v", "test")
}
