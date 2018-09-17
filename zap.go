package golog

import (
	"sync"

	"go.uber.org/zap"
)

var (
	zapConfig = zap.NewProductionConfig()
	zlog      *zap.SugaredLogger
	once      sync.Once
)

// SetZapConfig allows users to customize the configuration. Note this must be called before
// any logging takes place -- it will not reset the configuration of an existing logger.
func SetZapConfig(config zap.Config) {
	zapConfig = config
}

// ZapLogger creates a new zap logger with the specified name.
func ZapLogger(prefix string) *zap.SugaredLogger {
	once.Do(func() {
		baseLog, _ := zapConfig.Build()
		// Make sure our wrapper code isn't what always shows up as the caller.
		zlog = baseLog.Sugar()
	})
	return zlog.Named(prefix)
}
