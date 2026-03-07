package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

func New(w io.Writer, level Level) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: parseLevel(level)}))
}

func Default(level string) *slog.Logger {
	return New(os.Stderr, Level(strings.ToUpper(strings.TrimSpace(level))))
}

func WithIssue(logger *slog.Logger, issueID, issueIdentifier string) *slog.Logger {
	if logger == nil {
		logger = Default("")
	}
	return logger.With(
		slog.String("issue_id", issueID),
		slog.String("issue_identifier", issueIdentifier),
	)
}

func WithSession(logger *slog.Logger, sessionID string) *slog.Logger {
	if logger == nil {
		logger = Default("")
	}
	return logger.With(slog.String("session_id", sessionID))
}

func parseLevel(level Level) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(string(level))) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
