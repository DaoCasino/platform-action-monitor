package main

import (
	"fmt"

	"go.uber.org/zap"
)

var mainLog = zap.NewNop()
var sessionLog = zap.NewNop()
var scraperLog = zap.NewNop()
var methodLog = zap.NewNop()

var loggingEnabled = false

func EnableDebugLogging(l *zap.Logger) {
	mainLog = l
	sessionLog = l
	scraperLog = l
	methodLog = l

	loggingEnabled = true
}

func EnableMethodLogging() {
	methodLog = newLogger(false)
	enableLogging(methodLog)
}

func EnableMainLogging() {
	mainLog = newLogger(false)
	enableLogging(mainLog)
}

func EnableSessionLogging() {
	sessionLog = newLogger(false)
	enableLogging(sessionLog)
}

func EnableScraperLogging() {
	scraperLog = newLogger(false)
	enableLogging(scraperLog)
}

func enableLogging(logger *zap.Logger) {
	if loggingEnabled == false {
		logger.Warn("Enabling logs. Expect performance hits for high throughput")
		loggingEnabled = true
	}
}

type logStringerFunc func() string

func (f logStringerFunc) String() string { return f() }

func typeField(field string, v interface{}) zap.Field {
	return zap.Stringer(field, logStringerFunc(func() string {
		return fmt.Sprintf("%T", v)
	}))
}

func newLogger(production bool) (l *zap.Logger) {
	if production {
		l, _ = zap.NewProduction()
	} else {
		l, _ = zap.NewDevelopment()
	}
	return
}

// NewLogger a wrap to newLogger
func NewLogger(production bool) *zap.Logger {
	return newLogger(production)
}