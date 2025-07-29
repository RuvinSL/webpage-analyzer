package logger

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
)

// New creates a new structured logger that implements interfaces.Logger
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

// WithContext creates a logger with context values
func WithContext(ctx context.Context, logger interfaces.Logger) interfaces.Logger {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return logger.With(slog.String("request_id", requestID))
	}
	return logger
}

// WithError adds an error to the logger
func WithError(logger interfaces.Logger, err error) interfaces.Logger {
	if err != nil {
		return logger.With(slog.String("error", err.Error()))
	}
	return logger
}

// LoggerAdapter implements interfaces.Logger using slog
type LoggerAdapter struct {
	logger *slog.Logger
}

// NewAdapter creates a new logger adapter
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

// package logger

// import (
// 	"context"
// 	"log/slog"
// 	"os"
// 	"runtime"
// 	"time"

// 	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
// )

// // Logger wraps slog.Logger to implement our interface
// type Logger struct {
// 	*slog.Logger
// }

// // New creates a new structured logger
// func New(service string, level slog.Level) *slog.Logger {
// 	opts := &slog.HandlerOptions{
// 		Level: level,
// 		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
// 			// Customize time format
// 			if a.Key == slog.TimeKey {
// 				return slog.Attr{
// 					Key:   a.Key,
// 					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339)),
// 				}
// 			}
// 			return a
// 		},
// 	}

// 	handler := slog.NewJSONHandler(os.Stdout, opts)

// 	logger := slog.New(handler).With(
// 		slog.String("service", service),
// 		slog.Int("pid", os.Getpid()),
// 		slog.String("go_version", runtime.Version()),
// 	)

// 	return logger
// }

// // WithContext creates a logger with context values
// func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
// 	// Extract request ID from context if available
// 	if requestID, ok := ctx.Value("request_id").(string); ok {
// 		return logger.With(slog.String("request_id", requestID))
// 	}
// 	return logger
// }

// // WithError adds an error to the logger
// func WithError(logger *slog.Logger, err error) *slog.Logger {
// 	if err != nil {
// 		return logger.With(slog.String("error", err.Error()))
// 	}
// 	return logger
// }

// // LoggerAdapter adapts slog.Logger to implement interfaces.Logger
// type LoggerAdapter struct {
// 	logger *slog.Logger
// }

// // NewAdapter creates a new logger adapter
// func NewAdapter(logger *slog.Logger) interfaces.Logger {
// 	return &LoggerAdapter{logger: logger}
// }

// // Debug logs at debug level
// func (l *LoggerAdapter) Debug(msg string, args ...any) {
// 	l.logger.Debug(msg, args...)
// }

// // Info logs at info level
// func (l *LoggerAdapter) Info(msg string, args ...any) {
// 	l.logger.Info(msg, args...)
// }

// // Warn logs at warn level
// func (l *LoggerAdapter) Warn(msg string, args ...any) {
// 	l.logger.Warn(msg, args...)
// }

// // Error logs at error level
// func (l *LoggerAdapter) Error(msg string, args ...any) {
// 	l.logger.Error(msg, args...)
// }

// // With creates a new logger with additional fields
// func (l *LoggerAdapter) With(args ...any) interfaces.Logger {
// 	return &LoggerAdapter{
// 		logger: l.logger.With(args...),
// 	}
// }
