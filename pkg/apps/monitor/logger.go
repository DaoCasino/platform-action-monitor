package monitor

import (
	"go.uber.org/zap"
)

var mainLog = zap.NewNop()
var sessionLog = zap.NewNop()
var scraperLog = zap.NewNop()
var methodLog = zap.NewNop()
var decoderLog = zap.NewNop()

func EnableDebugLogging(l *zap.Logger) {
	mainLog = l
	sessionLog = l
	scraperLog = l
	methodLog = l
	decoderLog = l
}

func newLogger(production bool) (l *zap.Logger) {
	if production {
		l, _ = zap.NewProduction()
	} else {
		l, _ = zap.NewDevelopment()
	}
	return
}
