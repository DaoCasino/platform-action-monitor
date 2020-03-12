package main

import (
	"go.uber.org/zap"
)

func init() {
	logger, _ := zap.NewDevelopment()
	EnableDebugLogging(logger)
}
