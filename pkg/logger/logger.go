package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
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

func NewWithFiles(service string, level slog.Level, logDir string) interfaces.Logger {
	fmt.Printf("=== NewWithFiles DEBUG ===\n")
	fmt.Printf("Service: %s\n", service)
	fmt.Printf("LogDir: %s\n", logDir)

	// Create log directory if it doesn't exist
	fmt.Printf("Creating directory: %s\n", logDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create directory: %v\n", err)
		fmt.Printf("Falling back to stdout-only logger\n")
		return New(service, level)
	}
	fmt.Printf("Directory created/exists\n")

	// Create log file
	logFile := filepath.Join(logDir, service+".log")
	fmt.Printf("Creating log file: %s\n", logFile)

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to create log file: %v\n", err)
		fmt.Printf("Falling back to stdout-only logger\n")
		return New(service, level)
	}
	fmt.Printf("Log file created/opened\n")

	// Create multi-writer (both stdout and file)
	multiWriter := io.MultiWriter(os.Stdout, file)
	fmt.Printf("Multi-writer created (stdout + file)\n")
	fmt.Printf("==========================\n")

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

	handler := slog.NewJSONHandler(multiWriter, opts)

	baseLogger := slog.New(handler).With(
		slog.String("service", service),
		slog.Int("pid", os.Getpid()),
		slog.String("go_version", runtime.Version()),
	)

	return NewAdapter(baseLogger)
}

// // NewWithFiles creates a logger that writes to both stdout and files
// func NewWithFiles(service string, level slog.Level, logDir string) interfaces.Logger {
// 	// Create log directory if it doesn't exist
// 	if err := os.MkdirAll(logDir, 0755); err != nil {
// 		// Fallback to stdout only if we can't create log directory
// 		return New(service, level)
// 	}

// 	// Create log file
// 	logFile := filepath.Join(logDir, service+".log")
// 	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
// 	if err != nil {
// 		// Fallback to stdout only if we can't create log file
// 		return New(service, level)
// 	}

// 	// Create multi writer (both stdout and file)
// 	multiWriter := io.MultiWriter(os.Stdout, file)

// 	opts := &slog.HandlerOptions{
// 		Level: level,
// 		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
// 			if a.Key == slog.TimeKey {
// 				return slog.Attr{
// 					Key:   a.Key,
// 					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339)),
// 				}
// 			}
// 			return a
// 		},
// 	}

// 	handler := slog.NewJSONHandler(multiWriter, opts)

// 	baseLogger := slog.New(handler).With(
// 		slog.String("service", service),
// 		slog.Int("pid", os.Getpid()),
// 		slog.String("go_version", runtime.Version()),
// 	)

// 	return NewAdapter(baseLogger)
// }

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
