// Package logger provides a single place to configure and swap the logging
// backend. All packages call logger.Info/Warn/Error/Debug; the underlying
// implementation (slog, zap, Datadog, Loki, …) lives here and nowhere else.
package logger

import (
	"log/slog"
	"os"
)

// Level controls which messages are emitted.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Format controls the output encoding.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Options configures the logger. Zero value produces text output at Info level.
type Options struct {
	Level  Level
	Format Format
}

var l *slog.Logger

func init() {
	// Sensible default so the logger works even if Init is never called.
	l = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// Init configures the global logger. Call once from main before any logging.
func Init(opts Options) {
	ho := &slog.HandlerOptions{Level: toSlogLevel(opts.Level)}

	var handler slog.Handler
	if opts.Format == FormatJSON {
		handler = slog.NewJSONHandler(os.Stderr, ho)
	} else {
		handler = slog.NewTextHandler(os.Stderr, ho)
	}

	l = slog.New(handler)
	slog.SetDefault(l)
}

func toSlogLevel(level Level) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func Info(msg string, args ...any)  { l.Info(msg, args...) }
func Warn(msg string, args ...any)  { l.Warn(msg, args...) }
func Error(msg string, args ...any) { l.Error(msg, args...) }
func Debug(msg string, args ...any) { l.Debug(msg, args...) }
