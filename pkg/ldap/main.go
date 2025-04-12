package ldap

import (
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

// Initialize needs to be run to initialize this modules logger before usage
func Initialize(sugar *zap.SugaredLogger) {
	log = sugar
}
