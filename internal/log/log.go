package log

import (
	"io"
	"log/slog"
)

type ILogger interface {
	SetLevel(level Level)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Opts func(*Logger)

type Logger struct {
	// ILogger
	leveler *slog.LevelVar
	w       *slog.Logger
}

func New(w io.Writer, opts ...Opts) *Logger {
	leveler := &slog.LevelVar{}
	leveler.Set(slog.LevelError)
	l := &Logger{
		leveler: leveler,
		w:       slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: leveler})),
	}
	return l
}

type Level string

func (lvl Level) getSlogLevel() slog.Level {
	switch lvl {
	case "debug":
		return slog.LevelDebug
	case "error":
		return slog.LevelError
	case "info":
		return slog.LevelInfo
	}
	return 8
}

const (
	DebugLvl Level = "debug"
	InfoLvl  Level = "info"
	ErrorLvl Level = "error"
)

func (l *Logger) SetLevel(level Level) {
	l.leveler.Set(level.getSlogLevel())
}

func (l *Logger) Debug(msg string, args ...any) {
	l.w.Debug(msg, args...)
}
func (l *Logger) Error(msg string, args ...any) {
	l.w.Error(msg, args...)

}
func (l *Logger) Info(msg string, args ...any) {
	l.w.Info(msg, args...)
}
