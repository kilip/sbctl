package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type slogLogger struct {
	handler slog.Logger
}

// NewSlogLogger creates a new Logger implementation using slog.
func NewSlogLogger(level string) Logger {
	return NewSlogLoggerWithWriter(level, os.Stdout)
}

// NewSlogLoggerWithWriter creates a new Logger implementation using slog with custom writer.
func NewSlogLoggerWithWriter(level string, w io.Writer) Logger {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	// Use TextHandler for CLI, could switch to JSONHandler if needed
	handler := slog.New(slog.NewTextHandler(w, opts))

	return &slogLogger{
		handler: *handler,
	}
}

func (l *slogLogger) Debug(msg string, fields ...any) {
	l.handler.Debug(msg, fields...)
}

func (l *slogLogger) Info(msg string, fields ...any) {
	l.handler.Info(msg, fields...)
}

func (l *slogLogger) Warn(msg string, fields ...any) {
	l.handler.Warn(msg, fields...)
}

func (l *slogLogger) Error(msg string, fields ...any) {
	l.handler.Error(msg, fields...)
}

func (l *slogLogger) Fatal(msg string, fields ...any) {
	l.handler.Error(msg, fields...)
	os.Exit(1)
}

func (l *slogLogger) Panic(msg string, fields ...any) {
	l.handler.Error(msg, fields...)
	panic(msg)
}

func (l *slogLogger) DebugContext(ctx context.Context, msg string, fields ...any) {
	l.handler.DebugContext(ctx, msg, fields...)
}

func (l *slogLogger) InfoContext(ctx context.Context, msg string, fields ...any) {
	l.handler.InfoContext(ctx, msg, fields...)
}

func (l *slogLogger) WarnContext(ctx context.Context, msg string, fields ...any) {
	l.handler.WarnContext(ctx, msg, fields...)
}

func (l *slogLogger) ErrorContext(ctx context.Context, msg string, fields ...any) {
	l.handler.ErrorContext(ctx, msg, fields...)
}

func (l *slogLogger) With(fields ...any) Logger {
	return &slogLogger{
		handler: *l.handler.With(fields...),
	}
}
