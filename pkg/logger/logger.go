package logger

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
)

func New(service string, level slog.Level) interfaces.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339)),
				}
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)

	baseLogger := slog.New(handler).With(
		slog.String("service", service),
		slog.Int("pid", os.Getpid()),
		slog.String("go_version", runtime.Version()),
	)

	return NewAdapter(baseLogger)
}

func WithContext(ctx context.Context, logger interfaces.Logger) interfaces.Logger {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return logger.With(slog.String("request_id", requestID))
	}
	return logger
}

func WithError(logger interfaces.Logger, err error) interfaces.Logger {
	if err != nil {
		return logger.With(slog.String("error", err.Error()))
	}
	return logger
}

type LoggerAdapter struct {
	logger *slog.Logger
}

func NewAdapter(logger *slog.Logger) interfaces.Logger {
	return &LoggerAdapter{logger: logger}
}

func (l *LoggerAdapter) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *LoggerAdapter) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *LoggerAdapter) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *LoggerAdapter) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *LoggerAdapter) With(args ...any) interfaces.Logger {
	return &LoggerAdapter{
		logger: l.logger.With(args...),
	}
}
