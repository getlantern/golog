package golog

import (
	"testing"

	"go.uber.org/zap"

	"github.com/getlantern/errors"
	"github.com/getlantern/ops"
)

func TestZap(t *testing.T) {
	ConfigureZap(zap.NewProductionConfig())
	log := NewLogger("myprefix")
	log.Info("I'm starting")

	parent := ops.Begin("parent").Set("a", 1)
	defer parent.End()

	err := errors.New("Failed in parent").With("b", 2)

	child := ops.Begin("child").Set("b", 22).Set("c", 3)
	defer child.End()

	log.Error(err)
	log.Errorf("%v failed: %v", "Something", err)
	log.Errorw("It definitely failed", "error", err)
}
