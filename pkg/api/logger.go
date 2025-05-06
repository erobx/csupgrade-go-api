package api

import (
	"log/slog"
	"os"
)

type LogService interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type logService struct {
	logger *slog.Logger
}

func NewLogger() LogService {
	return &logService{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (l *logService) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *logService) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}
