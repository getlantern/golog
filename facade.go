package golog

import (
	"log"
	"sync/atomic"
)

var loggerBuilder atomic.Value

type baseLoggerBuilder func(prefix string, debugOn bool, printStack bool) baseLogger

func setBaseLoggerBuilder(builder baseLoggerBuilder) {
	loggerBuilder.Store(builder)
}

type loggerFacade struct {
	prefix         string
	isDebugEnabled bool
	printStack     bool
}

func (lf *loggerFacade) getBaseLogger() baseLogger {
	return loggerBuilder.Load().(baseLoggerBuilder)(lf.prefix, lf.isDebugEnabled, lf.printStack)
}

func (lf *loggerFacade) Info(arg interface{}) {
	lf.getBaseLogger().Info(arg)
}

func (lf *loggerFacade) Infof(msg string, args ...interface{}) {
	lf.getBaseLogger().Infof(msg, args...)
}

func (lf *loggerFacade) Infow(msg string, keysAndValues ...interface{}) {
	lf.getBaseLogger().Infow(msg, keysAndValues...)
}

func (lf *loggerFacade) Error(arg interface{}) error {
	return lf.getBaseLogger().Error(arg)
}

func (lf *loggerFacade) Errorf(msg string, args ...interface{}) error {
	return lf.getBaseLogger().Errorf(msg, args...)
}

func (lf *loggerFacade) Errorw(msg string, keysAndValues ...interface{}) error {
	return lf.getBaseLogger().Errorw(msg, keysAndValues...)
}

func (lf *loggerFacade) Fatal(arg interface{}) {
	lf.getBaseLogger().Fatal(arg)
}

func (lf *loggerFacade) Fatalf(msg string, args ...interface{}) {
	lf.getBaseLogger().Fatalf(msg, args...)
}

func (lf *loggerFacade) Fatalw(msg string, keysAndValues ...interface{}) {
	lf.getBaseLogger().Fatalf(msg, keysAndValues...)
}

func (lf *loggerFacade) Debug(arg interface{}) {
	lf.getBaseLogger().Debug(arg)
}

func (lf *loggerFacade) Debugf(msg string, args ...interface{}) {
	lf.getBaseLogger().Debugf(msg, args...)
}

func (lf *loggerFacade) Debugw(msg string, keysAndValues ...interface{}) {
	lf.getBaseLogger().Debugw(msg, keysAndValues...)
}

func (lf *loggerFacade) AsStdLogger() *log.Logger {
	return lf.getBaseLogger().AsStdLogger()
}

func (lf *loggerFacade) IsDebugEnabled() bool {
	return lf.isDebugEnabled
}
