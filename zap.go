package golog

import (
	"errors"
	"fmt"
	"log"

	"go.uber.org/zap"
)

type zapper struct {
	z *zap.SugaredLogger
}

// ZapConfig is the public zap configuration.
var ZapConfig = zap.NewProductionConfig()

func zapLogger(prefix string) Logger {
	logger, _ := ZapConfig.Build()

	// Make sure our wrapper code isn't what always shows up as the caller.
	logger = logger.WithOptions(zap.AddCallerSkip(1))
	return &zapper{z: logger.Named(prefix).Sugar()}
}

func (z *zapper) AsStdLogger() *log.Logger {
	return zap.NewStdLog(z.z.Desugar())
}

func (z *zapper) Debug(arg interface{}) {
	z.z.Debug(fmt.Sprintf("%v", arg))
}

func (z *zapper) Debugf(message string, args ...interface{}) {
	z.z.Debugf(message, args...)
}

func (z *zapper) Info(arg interface{}) {
	z.z.Info(fmt.Sprintf("%v", arg))
}

func (z *zapper) Infof(message string, args ...interface{}) {
	z.z.Infof(message, args...)
}

func (z *zapper) Error(arg interface{}) error {
	msg := fmt.Sprintf("%v", arg)
	z.z.Error(msg)
	return errors.New(msg)
}

func (z *zapper) Errorf(message string, args ...interface{}) error {
	msg := fmt.Sprintf(message, args...)
	z.z.Errorf(message, args)
	return errors.New(msg)
}

func (z *zapper) Fatal(arg interface{}) {
	z.z.Fatal(fmt.Sprintf("%v", arg))
}

func (z *zapper) Fatalf(message string, args ...interface{}) {
	z.z.Fatalf(message, args...)
}
