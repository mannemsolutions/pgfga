package pg

import (
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

// Initialize needs to be run to initialize this modules logger before usage
func Initialize(logger *zap.SugaredLogger) {
	log = logger
}
