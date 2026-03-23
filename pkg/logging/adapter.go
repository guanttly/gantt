package logging

import (
	"github.com/go-kit/log"
)

type SlogKitLogger struct {
	logger ILogger
}

func NewSlogKitLogger(logger ILogger) log.Logger {
	return &SlogKitLogger{logger: logger}
}

func (l *SlogKitLogger) Log(keyVals ...any) error {
	l.logger.Info("Go-Kit Log", keyVals...)
	return nil
}
