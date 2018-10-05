package golog

import (
	"fmt"
	"log"
	"sync"

	"go.uber.org/zap"

	"github.com/getlantern/errors"
	"github.com/getlantern/hidden"
	"github.com/getlantern/ops"
)

// ConfigureZap configures golog to use a Zap backend as configured with the given zap.Config
func ConfigureZap(cfg zap.Config) {
	var structuredLoggerInstances sync.Map
	setBaseLoggerBuilder(func(prefix string, debugOn bool, printStack bool) baseLogger {
		structuredLogger, found := structuredLoggerInstances.Load(prefix)
		if !found {
			stacktraceLevel := zap.ErrorLevel
			if isStackEnabled() {
				stacktraceLevel = zap.DebugLevel
			}
			// TODO: figure out how to control log level (e.g. with TRACE flag or something else)
			logger, err := cfg.Build(zap.AddStacktrace(stacktraceLevel))
			if err != nil {
				fmt.Printf("Error configuring Zap logger, will use stream logger: %v\n", err)
				structuredLogger = &streamLogger{
					prefix:     prefix + ": ",
					debugOn:    debugOn,
					printStack: printStack,
				}
			} else {
				structuredLogger = &zapLogger{logger.Sugar()}
			}
			structuredLoggerInstances.Store(prefix, structuredLogger)
		}
		return structuredLogger.(baseLogger)
	})
}

type zapLogger struct {
	*zap.SugaredLogger
}

func (l *zapLogger) Debug(arg interface{}) {
	l.getSugaredLogger(nil).Debug(hidden.Clean(fmt.Sprint(arg)))
}

func (l *zapLogger) Debugf(template string, args ...interface{}) {
	l.getSugaredLogger(nil).Debug(hidden.Clean(fmt.Sprintf(template, args...)))
}

func (l *zapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.getSugaredLogger(nil).Debugw(msg, keysAndValues...)
}

func (l *zapLogger) Info(arg interface{}) {
	l.getSugaredLogger(nil).Info(hidden.Clean(fmt.Sprint(arg)))
}

func (l *zapLogger) Infof(template string, args ...interface{}) {
	l.getSugaredLogger(nil).Infof(hidden.Clean(fmt.Sprintf(template, args...)))
}

func (l *zapLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.getSugaredLogger(nil).Infow(msg, keysAndValues...)
}

func (l *zapLogger) Error(arg interface{}) error {
	err := l.getError("%v", arg)
	l.getSugaredLogger(err).Error(hidden.Clean(fmt.Sprint(arg)))
	return err
}

func (l *zapLogger) Errorf(template string, args ...interface{}) error {
	err := l.getError(template, args...)
	l.getSugaredLogger(err).Errorf(hidden.Clean(fmt.Sprintf(template, args...)))
	return err
}

func (l *zapLogger) Errorw(msg string, keysAndValues ...interface{}) error {
	err := l.getError(msg, keysAndValues...)
	l.getSugaredLogger(err).Errorw(msg, keysAndValues...)
	return err
}

func (l *zapLogger) Fatal(arg interface{}) {
	err := l.getError("%v", arg)
	l.getSugaredLogger(err).Fatal(hidden.Clean(fmt.Sprint(arg)))
	fatal(err)
}

func (l *zapLogger) Fatalf(template string, args ...interface{}) {
	err := l.getError(template, args...)
	l.getSugaredLogger(err).Fatalf(hidden.Clean(fmt.Sprintf(template, args...)))
	fatal(err)
}

func (l *zapLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	err := l.getError(msg, keysAndValues...)
	l.getSugaredLogger(err).Fatalw(msg, keysAndValues...)
	fatal(err)
}

func (l *zapLogger) AsStdLogger() *log.Logger {
	return zap.NewStdLog(l.getSugaredLogger(nil).Desugar())
}

func (l *zapLogger) getSugaredLogger(err error) *zap.SugaredLogger {
	sl := l.SugaredLogger
	ctx := ops.AsMap(err, false)
	for key, value := range ctx {
		sl = sl.With(zap.Any(key, value))
	}
	return sl
}

func (l *zapLogger) getError(template string, args ...interface{}) error {
	for _, arg := range args {
		switch e := arg.(type) {
		case error:
			return e
		}
	}
	return errors.NewOffset(2, template, args...)
}

type zapStdLogger struct {
}
